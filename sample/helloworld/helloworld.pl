
// 写入配置块，设置路由
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
