config service {
  .name = "concate";
  .router = "[GET]/concate";
  application concate();
}

rule "concate.generate" => {
  let separator = "\n\n\n\n<!---- SEPARATOR FROM MONO SERVICE ---->\n\n\n\n";

  output => [
    http::new_url("https://www.tmall.com"), 
    separator,
    http::new_request("POST", "https://www.taobao.com", "Hello World")
  ];
}

rule "concate.background.check" => {
  pass => true;
}

// start generate output
rule "concate.response" => {
  response.body = $.output;
  response.status = 201;
}
