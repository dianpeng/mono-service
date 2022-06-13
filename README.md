# Mono Service

Mono Service is a monolithical application server that tries to be an busy box
for application logic. It tries to provide lots of common task and work out of
the box to simplify SOA deployment. 

The server itself has an extreamly extensive architecture, with lots of builtin
modules written in native Go. And the server provides a highly integrated and
total innovative scripting language to work around the modification of behavior
of runtime behavior.


# Features

## Extensiable Pluggin Style Module API

MonoService can be easily extending its functionality with its core APIs in go.
Although, there will be lots of function builtin, user still can use the Go api
to create specific function to extend Mono Service's runtime.


## Specialized scripting DSL

MonoService runtime features a entirely designed from scratch scripting engine
called PL, ie Policy Language. Unlike most web server tries to integrate/embed
a general purpose language. Our PL is desigend from scratch to support Http or
application logic dynamic configuration. The langauge supports a rule based 
code orgnization to catch different runtime event happened, also its variable has
session awareness. Certain types of variable is awaring of http session and will
be reset automatically when each http session are gone and hold its value during
one exact http session. Additionally, it supports template builtin. User can
render go template, markdown template directly inside of the code. The runtime
also allows addition of other template engine if user wants to use. 


## Multi Tenant Awareness

The MonoService runtime is desigened with multi tenancy in mind. The service
exposed by each HTTP endpoint can be configured flexibly by configuration file.
Additionally, each group of services been exposed can be groupped together to
form a virtual host. A vhost is been used as an representation of tenancy.


## Sample

sample/sample.yaml


## Status

Very early stage of development, stay tuned :)

