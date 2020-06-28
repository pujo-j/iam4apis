package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dgraph-io/ristretto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pujo-j/iam4apis/graph/model"
	"net/http"
	"sync"
	"time"
)

type PostgreStore struct {
	pool          *pgxpool.Pool
	cache         *ristretto.Cache
	eventTail     []*model.AdminEvent
	eventTailLock sync.RWMutex
	eventNotif    chan struct{}
}

func NewPostgreStore(url string) (*PostgreStore, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e6,
		MaxCost:     1000,
		BufferItems: 64, // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(context.Background(), "SELECT email FROM users")
	if err != nil {
		_, err = pool.Exec(context.Background(), `
			CREATE TABLE public.users
			(
				email character varying COLLATE pg_catalog."default" NOT NULL,
				active boolean NOT NULL,
				"fullName" character varying COLLATE pg_catalog."default",
				profile character varying COLLATE pg_catalog."default",
				"firstAccess" timestamp with time zone,
				"lastAccess" timestamp with time zone,
				roles jsonb,
				CONSTRAINT users_pkey PRIMARY KEY (email)
			)`)
		if err != nil {
			return nil, err
		}
		_, err = pool.Exec(context.Background(), `
			CREATE TABLE public.admin_events
			(
				id bigserial NOT NULL,
				ts timestamp with time zone NOT NULL,
				admin_id character varying NOT NULL,
				user_id character varying NOT NULL,
				roles jsonb NOT NULL,
				PRIMARY KEY (id),
				CONSTRAINT admin_fk FOREIGN KEY (admin_id)
					REFERENCES public.users (email) MATCH SIMPLE,
				CONSTRAINT user_fk FOREIGN KEY (user_id)
					REFERENCES public.users (email) MATCH SIMPLE
			);`)
		if err != nil {
			return nil, err
		}
	} else {
		rows.Close()
	}
	res := &PostgreStore{
		pool:       pool,
		cache:      cache,
		eventTail:  make([]*model.AdminEvent, 0, 50),
		eventNotif: make(chan struct{}, 0),
	}
	go res.PollEvents()
	return res, nil
}

func (p *PostgreStore) GetUser(ctx context.Context, email string) (*model.User, error) {
	user, found := p.cache.Get(email)
	if found {
		return user.(*model.User), nil
	}
	res := model.User{}
	var rolesJsons string
	err := p.pool.QueryRow(ctx, "SELECT email, active, \"fullName\", profile, \"firstAccess\", \"lastAccess\", roles FROM users WHERE email=$1", email).Scan(
		&res.Email,
		&res.Active,
		&res.FullName,
		&res.Profile,
		&res.FirstAccess,
		&res.LastAccess,
		&rolesJsons,
	)
	if err != nil {
		return nil, err
	}
	res.Roles = make([]*model.Role, 0)
	err = json.Unmarshal([]byte(rolesJsons), &res.Roles)
	if err != nil {
		return nil, err
	}
	p.cache.Set(email, &res, 1)
	return &res, nil
}

func (p *PostgreStore) GetUsers(ctx context.Context, fromId *string) ([]*model.User, error) {
	var rows pgx.Rows
	var err error
	if fromId == nil {
		rows, err = p.pool.Query(ctx, "SELECT email, active, \"fullName\", profile, \"firstAccess\", \"lastAccess\", roles FROM users ORDER BY email LIMIT 100")
	} else {
		rows, err = p.pool.Query(ctx, "SELECT email, active, \"fullName\", profile, \"firstAccess\", \"lastAccess\", roles FROM users WHERE email>$1 ORDER BY email LIMIT 100", fromId)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := make([]*model.User, 0, 100)
	for rows.Next() {
		u := model.User{}
		var rolesJsons string
		err = rows.Scan(
			&u.Email,
			&u.Active,
			&u.FullName,
			&u.Profile,
			&u.FirstAccess,
			&u.LastAccess,
			&rolesJsons,
		)
		if err != nil {
			return nil, err
		}
		u.Roles = make([]*model.Role, 0)
		err = json.Unmarshal([]byte(rolesJsons), &u.Roles)
		if err != nil {
			return nil, err
		}
		res = append(res, &u)
	}
	return res, nil
}

func (p *PostgreStore) SearchUsers(ctx context.Context, namePart string) ([]*model.User, error) {
	rows, err := p.pool.Query(ctx, "SELECT email, active, \"fullName\", profile, \"firstAccess\", \"lastAccess\", roles FROM users WHERE \"fullName\" LIKE $1 OR email LIKE $2 LIMIT 100", namePart+"%", namePart+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := make([]*model.User, 0, 100)
	for rows.Next() {
		u := model.User{}
		var rolesJsons string
		err = rows.Scan(
			&u.Email,
			&u.Active,
			&u.FullName,
			&u.Profile,
			&u.FirstAccess,
			&u.LastAccess,
			&rolesJsons,
		)
		if err != nil {
			return nil, err
		}
		u.Roles = make([]*model.Role, 0)
		err = json.Unmarshal([]byte(rolesJsons), &u.Roles)
		if err != nil {
			return nil, err
		}
		res = append(res, &u)
	}
	return res, nil
}

func (p *PostgreStore) UpdateUser(ctx context.Context, newUser model.EditUser) (*model.User, error) {
	editor := ForContext(ctx)
	if editor != nil && editor.IsInRole("admin", "/") {
		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback(ctx)
			}
		}()
		roles, err := json.Marshal(newUser.Roles)
		_, err = tx.Exec(ctx, `
		INSERT INTO users(email,active,roles) VALUES($1,false,$2)
		ON CONFLICT ON CONSTRAINT users_pkey DO UPDATE SET roles=$2
`,
			newUser.Email,
			string(roles),
		)
		if err != nil {
			_ = tx.Rollback(ctx)
			return nil, err
		}
		_, err = tx.Exec(ctx, `
		INSERT INTO public.admin_events(
			ts, admin_id, user_id, roles)
			VALUES (now(), $1, $2, $3);
		`, editor.Email, newUser.Email, roles)
		if err != nil {
			_ = tx.Rollback(ctx)
			return nil, err
		}
		err = tx.Commit(ctx)
		p.eventNotif <- struct{}{}
		if err != nil {
			_ = tx.Rollback(ctx)
			return nil, err
		}
		return p.GetUser(ctx, newUser.Email)
	} else {
		return nil, errors.New("unauthorized")
	}
}

func (p *PostgreStore) GetEvents(ctx context.Context, from *time.Time) ([]*model.AdminEvent, error) {
	p.eventTailLock.RLock()
	defer p.eventTailLock.RUnlock()
	var from2 time.Time
	if from == nil {
		from2 = time.Now().Add(-24 * time.Hour)
	} else {
		from2 = *from
	}
	res := make([]*model.AdminEvent, 0, 50)
	if len(p.eventTail) > 0 {
		firstEvent := p.eventTail[0]
		if from2.After(firstEvent.Ts) {
			for _, event := range p.eventTail {
				if event.Ts.After(from2) {
					res = append(res, event)
				}
			}
			return res, nil
		}
	}
	rows, err := p.pool.Query(ctx, "SELECT CAST(id as CHARACTER VARYING) as id, ts, admin_id, user_id, roles FROM admin_events WHERE ts>$1 ORDER BY ts LIMIT 50", from2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		event := model.AdminEvent{}
		var rolesJsons string
		err = rows.Scan(&event.ID, &event.Ts, &event.AdminID, &event.UserID, &rolesJsons)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(rolesJsons), &event.Roles)
		if err != nil {
			return nil, err
		}
		res = append(res, &event)
	}
	return res, nil
}

func (p *PostgreStore) EnrichUser(ctx context.Context, email string, fullName string, profile string) (*model.User, error) {
	currentUser := ForContext(ctx)
	if !currentUser.IsInRole("admin", "/") {
		if currentUser.Email != email {
			return nil, errors.New("cannot modify user")
		}
	}
	user, err := p.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}
	tag, err := p.pool.Exec(ctx, "UPDATE users set active=true, \"fullName\"=$2, profile=$3 WHERE email=$1", email, fullName, profile)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, errors.New("no such user")
	}
	user.FullName = &fullName
	user.Profile = &profile
	p.cache.Set(email, user, 1)
	return user, nil
}

func (p *PostgreStore) PollEvents() {
	bg := context.Background()
	var lastEventTime time.Time
	tick := time.Tick(5 * time.Second)
	var refresh = func() {
		rows, err := p.pool.Query(bg, "SELECT CAST(id as CHARACTER VARYING) as id, ts, admin_id, user_id, roles FROM admin_events WHERE ts>$1 ORDER BY ts LIMIT 50 ", lastEventTime)
		if err != nil {
			panic(err)
		}
		p.eventTailLock.Lock()
		for rows.Next() {
			event := model.AdminEvent{}
			var rolesJsons string
			err = rows.Scan(&event.ID, &event.Ts, &event.AdminID, &event.UserID, &rolesJsons)
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal([]byte(rolesJsons), &event.Roles)
			if err != nil {
				panic(err)
			}
			p.eventTail = append(p.eventTail, &event)
			if event.Ts.After(lastEventTime) {
				lastEventTime = event.Ts
			}
		}
		if len(p.eventTail) > 50 {
			p.eventTail = p.eventTail[len(p.eventTail)-49:]
		}
		p.eventTailLock.Unlock()
	}
	for {
		select {
		case <-tick:
			{
				refresh()
			}
		case <-p.eventNotif:
			{
				refresh()
			}
		}
	}
}

func (p *PostgreStore) MiddleWare(r *http.Request) context.Context {
	userHeader := r.Header.Get("X-User")
	if userHeader != "" {
		user, err := p.GetUser(r.Context(), userHeader)
		if err == nil {
			ctx := context.WithValue(r.Context(), userKey, user)
			return ctx
		}
	}
	return r.Context()
}

func (p *PostgreStore) Login(ctx context.Context) (*model.User, error) {
	user := ForContext(ctx)
	t := time.Now()
	if user == nil {
		return nil, errors.New("no such user")
	}
	if user.FirstAccess == nil {
		tag, err := p.pool.Exec(ctx, "UPDATE users SET \"firstAccess\"=$2, \"lastAccess\"=$2 WHERE email=$1", user.Email, t)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != 1 {
			return nil, errors.New("update user failed")
		}
		user.FirstAccess = &t
		user.LastAccess = &t
		p.cache.Set(user.Email, user, 1)
	} else {
		tag, err := p.pool.Exec(ctx, "UPDATE users SET \"lastAccess\"=$2 WHERE email=$1", user.Email, t)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != 1 {
			return nil, errors.New("update user failed")
		}
		user.LastAccess = &t
		p.cache.Set(user.Email, user, 1)
	}
	return user, nil
}
