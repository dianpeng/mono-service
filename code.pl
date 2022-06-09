session {
	a = 10;
	b = 20;
	c = 200 - 1 + 1;
}

fn HelloWorld() {
  return "hello world";
}

response => {
	let proxy_url = request.header["x-proxy-url"];
	let list_of_url = str::split(proxy_url, ';');
  body => "{{c.to_string()}}\n{{HelloWorld()}}\n\n\n";
	status => c;
  print("Hello World", c, a, b);
}

error => {
  dprint(phase, error);
}
