import (
  "assets/test/module/mod3.m"
)


test {
  assert::eq(mod3::zz::foo(), "bar");
}
