# IAP4APIS

Simple authorization server

## What

A simple data microservice exposing a very simple authorization model User-Role-Resources with a postgresql backend.


## Why

The original need was to expose user-facing applications on cloud infrastructure,
 while cloud-provider IAM systems permit fine-grained roles on managed resources, 
 there is no non-hacky way to reuse the IAM system for applicative roles.
 
In these cases, we already have a great integrated identity and authentication provider, but are missing the authorization service.

While there are many all-in-one open source authn systems, they all provide identity, authentication and authorization.

 