# MonoService

MonoService是个全新的实验性质的Web/Http服务器。该服务器主要聚焦于利用可编程脚本来聚合不同的Go编写的功能组件，从而快速的搭建一个Web/Http的接入，应用服务器。

本质上，MonoService提供如下主要功能组件：

1. 一个规规范化的HTTP请求/接入流程
	* 现代Web/Http服务器Framework
		* 服务发现
		* Hot Reloading
		* Middleware
		* 日志
		* 监控
		* 可编程
	* 流行的说法：Opinioned Framework

2. 一个从头设计实现的脚本编程语言，名叫PL(Policy Language)
	* 专门为Http/Web场景设计
		* Http/Web场景功能
		* 事件驱动
		* 多种编程范式支持
		* 转为Web服务器设计
	* 笔者多年工业Http/Web服务器研发经验的反省


3. 一个运行时框架
	* Go语言开发
	* 高度模块化
	* 内置大量覆盖常规场景的功能组件


另外一种视角为，MonoService是个编程语言，伪装在一个Web/Http服务器之后，专门用来结合Go提供Http流量的接入，业务功能。


## PL(策略语言)

PL是MonoServcice重新设计的专门为HTTP/Web的脚本语言，他支持多种先代编程语言的
特性和符合web场景的特点。

1. HTTP Transaction变量

  * 变量的生命周期和一次http请求回复一致

2. HTTP 全局变量

  * Immutable
  * 自动多个HTTP请求共享，线程安全

3. 内置配置
  * 配置块也是代码，用户可以在配置中使用任何语言特性，for，if，变量，函数等等

4. 丰富的脚本语言特性
  * 支持事件驱动触发
  * 高阶函数支持
    * Closure
    * 上值捕获
    * generator/yield

  * 控制流表达式
    * if/try/语法块可以当作表达式使用

  * try 风格异常处理
    * try也可以当表达式使用

  * 正则表达式常量
  * 字符串模版
  * 内置Web模版渲染引擎
    * go
    * mardkwon
    * pongo2

  * 丰富类型系统
    * 用户可以用go轻松扩展类型系统


[PL语言详情](doc/pl-zn.md)


## 功能

1. 现代Http/Web服务器功能
	1. Hot Reloading
	2. 服务发现
	3. 模块化
	4. 可编程
	5. Middleware组合
	6. k8生态监控
	7. 日志

2. 多租户
	1. 应用打包
	2. Virtual Host类型租户隔离

3. 0配置
	1. DSL代码支持配置，无需使用单独JSON/Yaml/TOML等配置文件
	2. 0配置快速开启服务


4. 高度灵活
	1. 代码定义的Web/Http服务器
	2. 和Nginx不同，不是在Web服务器中嵌入编程语言，而是整个Web服务器运行在一个专门为Web/Http设计的编程语言中



## 第一个例子


在MonoService中，每个租户对应一个称为虚拟Host/Virtual Host的逻辑概念。该Virtual Host由一个PL代码定义，加上若干个挂在不同路由下被动触发的f服务/Service组成。假设一个用户要使用MonoService，他需要：

	* 编写一个PL脚本，描述Virtual Host这个组建
	* 编写若干PL脚本，每个脚本描述一个挂载在某个Router下的Service


1. 编写一个入口程序，表示一个virtual host

	* 先创建一个目录，比如 ``` mkdir helloworld ```
	* 在改目录内创建文件: main.pl，写入如下内容

```

// 如下为PL语言的配置代码。PL语言内置配置功能。
// 注意的每个配置项的名字前面有个“."，这个是因为每个config也是个代码块，用户可以编写常规代码。常规代码的符号名字是不能由'.'开头的。

config virtual_host {
  println("start to configure virtual_host.name");
  .name = "first_service"; // service名字
  
  println("start to configure virtual_host.server_name");
  .server_name = "example.com"; // 该服务/租户的host名字，用于通过http请求host头触发
  
  println("start to configure virtual_host.listener");
  let listener_name = "test";
  .listener = listener_name;  // 该服务器挂载的listener名字。listener可以简单理解为监听的地址和端口号
  
}

```

2. 编写了程序入口，我们还需要为我们的租户添加一个具体的应用，即某个path的请求进入服务器下该租户，这个租户需要干些什么


	* 在 helloworld目录下，创建一个新的文件，名字为hello_world.pl，写入如下内容


```

// 写入配置块，设置每个路由下面的程序，配置名称为service
config service {
  .name = "helloworld"; // 这个路由下的程序的名字
  .router = "[GET]/greetings"; // 路由设置，表示GET请求下，路径为/greetings时候触发下面的程序
  
  // 配置该路由下的应用程序的流程逻辑
  // 每个请求进入到某个路由下的程序后，会经过如下处理流程
  // 1. Request阶段
  //    该阶段下，用户可以配置组合若干Request阶段的请求Middleware，比如JWT/Basic 鉴权，限流等内置Go模块，
  //    也可以触发客户利用PL脚本定制的Middleware逻辑
  
  // 2. Application阶段
  //    该阶段可以认为是处理这个HTTP请求的main函数，他就在所有的Reqeust阶段的Middeleware执行完毕后执行。
  //    这个代码可以用Go编写或者用PL。MonoService已经内置了一些常见的Application功能
  
  // 3. Response阶段
  //    该阶段为生成HTTP恢复的阶段。当Application阶段代码执行结束后，该Response阶段触发。用户可以组合多个
  //    不同的middleware用于回复生成，比如加解压缩，增加特殊回复头，或者触发自定义PL代码
 
  // 配置request阶段
  request {
    // 在reqeust阶段，我们不用任何go的内置request middleware，而是简单触发下我们自定义的PL代码
    .event("my_request");
  }
  
  // 配置 application 阶段
  // 因为我们hello_world没有任何复杂的逻辑需要执行，所以，我们使用内置的空操作(noop)。注意，application
  // 阶段，和response/request不同，只能配置唯一一个入口
  application noop();
  
  // 配置 response阶段
  // 同样，我们的程序比较简单，因此，不使用任何复杂的response middleware，我们利用event触发一个我们编写的
  // 自定义脚本
  response {
    .event("my_response");
  }
}

// 编写上述request/response配置触发的代码
// 在上述 request/response中，我们使用event middleware触发了代码测的代码。event middleware会发送事件给我们的脚本
// 我们的脚本内需要编写事件handler来接受事件完成相应。

// 1. my_request 事件hanlder，在request阶段被触发。
rule my_request {

   // 我们打印一个消息，什么都不做
   println("this is from my_request handler");
   
   // 我们检测下，我们的host名字
   println("the request host is {{request.host}}");
}

// 2. my_response 时间handler，在response阶段被触发，
//    在这个handler中，我们必须设置response，即HTTP回复，否则没有任何回复信息

rule my_response {
  println("This is in my response");
  
  response.status = 200; // 回复200状态码
  response.body = "Hello World, client is: {{request.remoteAddr}}"; // 回复HelloWorld，顺便打印客户端ip地址
  
}

```

3. 现在我们启动MonoService去执行刚才的租户的整个代码

	* 	MonoService支持利用ZIP打包的租户代码
	* 	这个例子，这个helloworld文件夹就是真个租户的代码包

运行 ``` monoservice --listener "test,:18080" --path helloworld/main.pl ```

* 上述命令的--listener 创建了一个名字叫做test的，绑定在 ":18080"的非TLS Http监听器，还记得我们在helloworld程序的main.pl 配置virtualhost的时候，制定了listener的名字为test。这个地方创建了这个listener。

* --path 指定了某个租户的代码包。--path可以制定多次，每次指向的代码包path都会当作某个租户的service挂在到monoservice服务器


4. 然后用户可以使用 ``` curl -H"host: example.com" "http://localhost:18080/greetings" ```来测试下结果



## Status

目前服务器还处在开发中，功能还不完善，stay tuned
