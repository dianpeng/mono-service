global {
  taobao = http::get("http://www.qq.com").body:string();
}

config service {
  .name = "proxy_taobao";
  .router = "[GET]/taobao/{a}/{b}";

  application noop();
  response event("response");
}

rule response {
  let sub_resp = http::get("https://www.sina.com.cn");
  let payload_buffer = sub_resp.body.stream:string();
  println(sub_resp.status);

  response.status = 200 if sub_resp.status == 200 else 404;
  response.body = payload_buffer;
}
