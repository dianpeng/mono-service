/**
 ** testing the scoping rules of each variable, ie symbol resolution,
 ** symbol binding etc ...
 **/
global {
  const_1 = 10;
  const_2 = 20;
  const_3 = 30;
}

session {
  sess_1 = 10;
  sess_2 = 20;
  sess_3 = 30;
}

fn qualify_session() {
  assert::eq(session::sess_1, 10);
  assert::eq(session::sess_2, 20);
  assert::eq(session::sess_3, 30);
}

fn qualify_const() {
  assert::eq(global::const_1, 10);
  assert::eq(global::const_2, 20);
  assert::eq(global::const_3, 30);
}

test {
  qualify_session();
  qualify_const();
}
