import (
  "assets/test/module/mod1.m"
  "assets/test/module/mod2.m"
)

test {
  assert::eq(mm::yy(), "Hello World");
  assert::eq(zz::Add(1, 2), 3);
  assert::eq(mm::a, zz::ag);
  assert::eq(mm::b, zz::bg);
  assert::eq(global::zz::ag, 10);
}
