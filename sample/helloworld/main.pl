// 如下为PL语言的配置代码。PL语言内置配置功能。
// 注意的每个配置项的名字前面有个“."，这个是因为每个config也是个代码块，用户可以编写常规代码。常规代码的符号名字是不能由'.'开头的。

config http_vhost {
  println("start to configure virtual_host.name");
  .name = "first_service"; // service名字
  
  println("start to configure virtual_host.server_name");
  .server_name = "example.com"; // 该服务/租户的host名字，用于通过http请求host头触发
  
  println("start to configure virtual_host.listener");
  let lname = "t" + "est";
  .listener = lname;  // 该服务器挂载的listener名字。listener可以简单理解为监听的地址和端口号
  
}
