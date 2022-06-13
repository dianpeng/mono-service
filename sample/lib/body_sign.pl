rule error => {
  dprint(phase, error);
  status => 503;
  body => "whatever shit: {{phase}} => {{error}}\n";
}

// sign policy, used for handling signing result
rule sign => {
  status => 200;
  header_set => ("x-body-sign-final-result", sign);
  header_set => ("x-body-sign-method", signMethod);

  // notes, we need to use signBody, since the original body has been
  // consumed up
  body => signBody;
};

// policy for handling verification result
// reject means the verification failed; pass means the verification
// passed
rule reject => {
  status => 404;
  header_set  => [
    ("x-body-sign-result", sign),
    ("x-body-sign-expect", signExpect)
  ];

  // notes, we need to use signBody, since the original body has been
  // consumed up
  body => signBody;
};

rule pass => {
  status => 202;
  header_set => [
    ("x-body-sign-result", sign),
    ("x-body-sign-expect", signExpect)
  ];
  body => signBody;
};

rule log => {
  dprint(logFormat);
}
