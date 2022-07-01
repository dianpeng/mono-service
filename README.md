# Mono Service

MonoService is a new experimental Web/Http server which focus on providing
opinioned programmable service. It bundles lots of builtin functions plus
a design from scratched new programming language for gluing them together. User
can picure MonoService as a runtime that happens to run just like web server.

# Features

## Feature Rich Web Server

1. Hot reloading user's application
2. Application pakage
3. Rich metrics
4. Zero Configuration
  1. The scripting language can also be used for configuration
5. Highly customizable
  1. Use go to provide all middleware, plugins
  2. Use scripting language to add new features quickly or glue the function to
     form highly customized and flexible workflow

## Highly Modular

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

## Opionioned Framework

MonoService categorize an web application as following way,

1. Highest level is a *virtual host* object. A server name must be provided to
   allow an http request comming in

2. Each virtual host can have multiple services, each *service* is just a bunch
   of logic, either via PL or Go, under a certain router

3. Each *service* has 3 distinct phases for processing a certain HTTP request.

   1. Request, which is a list of middleware forms a chain of responsibility and
      modify/mutate the incomming requests along the road. For example, user
      can use JWT/Oauth2 authentication middleware here.

   2. Application. Once the request middleware phase is done, it enters a single
      user selected application unit. An application is where the main logic
      of services lies in. Typically user can just use Go to write its own
      application if it is highly customized, or use a builtin Go application
      if applicable.

   3. After application done, a list of response middleware starts to generate
      the http response. For example, user can use compression response
      middleware here, or use other middleware to generate response header etc..

4. The most cool feature of our service is that, during any processing stages
   described above, user can use PL to emit an event to trigger a customized
   scripting code to do anything. And each *Request*, *Application* or *Response*
   can also be customize its own behavior with PL script. For example, any
   modules configuration parameter can become dynamic function call of PL
   script and evaluated for every http transaction. This is nearly impossible
   to achieve in any existed Http/Web server. Basically due to PL and its
   designed for HTTP transaction semantic, user can use PL it create any
   static or dynamic workflow for every HTTP transaction.

## Multi Tenancy Awareness

The MonoService runtime is desigened with multi tenancy in mind. Each virtual
host object is entirely contained.

# Configuration

## DSL (Policy Language)

See document for DSL features:

[Policy Language](doc/pl.md)
[PL编程语言](doc/pl-cn.md)


## Listeners

Runtime can be used to specify listener to be used by each application. The
runtime is almost zero config. 

# Status

The project is still in very early stage of development. Stay tunned :)
