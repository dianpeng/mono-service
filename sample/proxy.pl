global {
  taobao = http::get("https://www.taobao.com").body:string();
}

config service {
  .name = "proxy_taobao";
  .router = "[GET]/taobao";

  application noop();
  response event("response");
}

rule response {
  let proxy_url = request.header.x_proxy_url;
  let url = proxy_url == "" if "https://tmall.com" else proxy_url;
  let resp = taobao;
  let sub_resp = http::get("https://www.toutiao.com");
  let payload_buffer = sub_resp.body.stream:string();

  response.status = 200 if sub_resp.status == 200 else 404;
  response.body = resp;
}
