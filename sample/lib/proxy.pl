const {
  taobao = http::do(http::new_request("GET", "https://www.taobao.com")).body.string();
}

rule response {
  let proxy_url = request.header.x_proxy_url;
  let url = proxy_url == "" ? "https://tmall.com" : proxy_url;
  let resp = taobao;
  let sub_resp = http::get("https://www.toutiao.com");
  let payload_buffer = sub_resp.body.stream.string();

  body => if sub_resp.status == 200 {
    payload_buffer;
  } else {
    "the response status code is {{sub_resp.status}} which is not 200";
  };

  // if you want to see the content of the data, just print them out
  // print(payload_buffer);

  status => sub_resp.status;
}
