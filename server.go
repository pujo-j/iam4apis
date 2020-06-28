package main

import (
	"context"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/pujo-j/iam4apis/graph/model"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/pujo-j/iam4apis/graph"
	"github.com/pujo-j/iam4apis/graph/generated"
)

const defaultPort = "4300"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	db_url := os.Getenv("POSTGRES_URL")
	if db_url == "" {
		db_url = "postgresql://iam4apis:iam4apis@localhost/iam4apis"
	}
	admin_user := os.Getenv("ADMIN_USER")
	if admin_user == "" {
		log.Fatal("ADMIN_USER env is mandatory")
	}
	store, err := NewPostgreStore(db_url)
	if err != nil {
		log.Fatal(err)
	}
	admin, err := store.GetUser(context.Background(), admin_user)
	if err != nil {
		dummyAdmin := &model.User{
			Email:  admin_user,
			Active: true,
			Roles: []*model.Role{{
				Name: "admin",
				Path: "/",
			}},
		}
		admin, err = store.UpdateUser(context.WithValue(context.Background(), userKey, dummyAdmin), model.EditUser{
			Email: admin_user,
			Roles: []*model.EditRole{{
				Name: "admin",
				Path: "/",
			}},
		})
		if err != nil {
			log.Fatal("creating admin user: ", err)
		}
	}
	if !admin.IsInRole("admin", "/") {
		edit := admin.Edit()
		edit.Roles = append(edit.Roles, &model.EditRole{
			Name: "admin",
			Path: "/",
		})
		ctx := context.WithValue(context.Background(), userKey, admin)
		_, err = store.UpdateUser(ctx, *edit)
		if err != nil {
			log.Fatal(err)
		}
	}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		Db: store,
	}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
	http.HandleFunc("/graphql", func(writer http.ResponseWriter, request *http.Request) {
		ctx := store.MiddleWare(request)
		srv.ServeHTTP(writer, request.WithContext(ctx))
	})

	log.Printf("listening to http://0.0.0.0:%s/", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
