fn foo() : 10
fn bar() : "Hello World"

test {
  assert::eq(foo(), 10);
  assert::eq(bar(), "Hello World");

  let xx = fn(): "Yoyo";
  assert::eq(xx(), "Yoyo");

  assert::eq(|{
    "y":to_upper() + "O":to_lower() + "yo";
  }, xx());
}
