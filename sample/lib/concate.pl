rule generate => {
  let separator = "\n\n\n\n<!---- SEPARATOR FROM MONO SERVICE ---->\n\n\n\n";

  output => [
    http::new_url("https://www.tmall.com"), 
    separator,
    http::new_request("POST", "https://www.taobao.com", "Hello World")
  ]
}

// do nothing just
rule check => {
  pass => true;
}

// generate output
rule response => {
  body => output;
  status => 201;
}
