import (
  "assets/test/module/mod1.m"
  "assets/test/module/mod2.m"
)

test {
  assert::eq(mm::yy(), "Hello World");
  assert::eq(zz::Add(1, 2), 3);

  assert::eq(mm::mod1_session_a, zz::mod2_session_a);
  assert::eq(mm::mod1_session_b, zz::mod2_session_b);

  assert::eq(mm::mod1_global_a, zz::mod2_global_a);
  assert::eq(mm::mod1_global_b, zz::mod2_global_b);

  assert::eq(session::mm::mod1_session_a, zz::mod2_session_a);
  assert::eq(session::mm::mod1_session_b, zz::mod2_session_b);

  assert::eq(mm::mod1_session_a, session::zz::mod2_session_a);
  assert::eq(mm::mod1_session_b, session::zz::mod2_session_b);

  assert::eq(global::mm::mod1_global_a, zz::mod2_global_a);
  assert::eq(global::mm::mod1_global_b, zz::mod2_global_b);

  assert::eq(mm::mod1_global_a, global::zz::mod2_global_a);
  assert::eq(mm::mod1_global_b, global::zz::mod2_global_b);
}
