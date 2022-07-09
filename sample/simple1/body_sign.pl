config service {
  .name = "body_sign";
  .router = "[GET,POST]/body_sign/{op}/{method}";

  application body_sign();
}

// sign policy, used for handling signing result
rule "body_sign.sign" => {
  response.status = 200;
  response.header:set("x-body-sign-final-result", $.sign);
  response.header:set("x-body-sign-method", $.signMethod);
  response.body = $.signBody;
}

rule "body_sign.reject" => {
  response.status = 404;
  response.header:set("x-body-sign-result", $.sign);
  response.header:set("x-body-sign-expect", $.signExpect);
  response.body = $.signBody;
}

rule "body_sign.pass" => {
  response.status = 202;
  response.header:set("x-body-sign-result", $.sign);
  response.header:set("x-body-sign-expect", $.signExpect);
  response.body = $.signBody;
}
