# Mono Service

MonoService is a Http application server that can be used for different use cases.
User can picture it as a busybox but for Web/Http application services.

MonoService runtime tries to include as much common Web/Http business logic into
a single binary as possible. Additionally, it exposes a highly confiurable and
flexible way for user to customize it into any particular use cases. It can be
used to setup a application server in minute. A quick leaner can extend its
functionality reliably.

# Features

## Modular Architecture

MonoService can be easily extending its functionality with its core APIs in Go.
Although, there will be lots of builtin functions, many common use cases are
already been coverted. But user can still use the Go api to create specific
function to extend MonoService's runtime.

## Programmable Pipeline

MonoService runtime features a specifically designed from scratch scripting engine
called PL, or Policy Language. Unlike most web server tries to integrate or embbed
a general purpose language, like Lua. I decide to design a new language syntax to
support Http or application logic dynamic configuration. The langauge supports
rule based code dispatch to catch different runtime events. Additionally its
variable lifecycle has session semantic. Certain types of variable's lifecycle is
tied to the Http session and is visiable throughout one specific Http session.
Other feature like, markdown/go template literal, regex literal, rule dispatching,
etc ... Anyone who is familiar with C/Go/Rust style langauge can pick it up easily

## Multi Tenancy Awareness

The MonoService runtime is desigened with multi tenancy in mind. The service
exposed by each HTTP endpoint can be configured flexibly by configuration file.
Additionally, each group of services been exposed can be groupped together to
form a virtual host. A vhost is been used as an representation of tenancy.

# Configuration

## DSL (Policy Language)

See document for DSL features:

[Policy Language](doc/pl.md)

## Yaml

In general the configuration is written as Yaml which is a standard way in
Cloud Native environment. To address some limitation of yaml file, runtime extends
the yaml with some customized tag. Notes, all customized tag should be written
as 

```
!inc "./my-file.yaml"
!inc_string "./my-string.txt"
!env "PATH"
!eval "http::do('http://www.cdn.com/example.yaml', 'GET')"
```

### inc path

Includes a path specified yaml file and parsed as if the content is pasted at
the exact position

### inc_string path

Includes a path specified file's content as string into the original yaml

### env name

Return environment variable with *name* specified

### eval expression

Evaluate a PL expression and return its result as string into the yaml


## Sample

User can check sample configuration located at sample/sample.yaml to start. 

## Status

The project is still in very early stage of development. Stay tunned :)
