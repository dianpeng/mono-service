const {
  taobao = http::do(
    http::new_request("GET", "https://www.taobao.com")
  ).body:string();
}

rule response {
  let proxy_url = request.header.x_proxy_url;
  let url = proxy_url == "" if "https://tmall.com" else proxy_url;
  let resp = taobao;
  let sub_resp = http::get("https://www.toutiao.com");
  let payload_buffer = sub_resp.body.stream:string();

  let i = 0;

  for {
    i = i + 1;
    if i >= 201 {
      break;
    }
  }

  body => "done";
  status => i;
}
