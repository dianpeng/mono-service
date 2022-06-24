fn foo() {
  return bar();
}

fn bar() {
  return var();
}

fn var() {
  return voo();
}

fn voo() {
  let a = 10;
  let b = 20;
  return doo(a, b);
}

fn doo(a, b) {
  let x = 10;
  let yy = 20;
  return voodoo(a, x, b, yy);
}

fn voodoo(a, b, c, d) {
  return a - b + c - d;
}

fn fib(a) {
  if a == 0 {
    return 1;
  }
  if a <= 2 {
    return a;
  }
  return fib(a-1) + fib(a-2);
}

fn done() {
  let x = 0;
  return x;
}

fn done2(a, b) {
  return a + b;
}

fn testBasic() {
  assert::throw(
    fn() {
      done(1, 2, 3);
    },
    "more arguments # than we need"
  );
  assert::throw(
    fn() {
      done2(1);
    },
    "less arguments # than we need"
  );
  assert::eq(foo(), 0);
  assert::eq(fib(0), 1);
  assert::eq(fib(1), 1);
  assert::eq(fib(2), 2);
  assert::eq(fib(3), fib(2) + fib(1));
  assert::eq(fib(10), fib(9) + fib(8));
  assert::eq(fib(15), fib(14) + fib(13));
  assert::eq(fib(20), fib(19) + fib(18));
  assert::eq(fib(30), fib(29) + fib(28));
}

// -----------------------------------------------------------------------------
// anonymous function and shortcut function invocatino
fn callfunc() {
  assert::eq((fn(): 1)(), 1);
  assert::eq(fn() { return 1; }(), 1);
  assert::eq((fn() { return 1; })(), 1);
}

// -----------------------------------------------------------------------------
// testing interleave frame with native function frame etc ...

fn inter1() {
  let foo = fn(): "Hello World";
  assert::eq(
    callback(
      fn() : callback(
        fn(): callback(
          fn(): callback(
            fn(): foo()
          )
        )
      )
    ), 
    foo()
  );
}

fn inter2() {
  let z = callback(
    return_something
  );

  assert::eq(z, "Hello");
}

fn inter3() {
  let z = callback(
    time::unix
  );
  assert::eq(z, time::unix());
}

fn testInter() {
  inter1();
  inter2();
  inter3();
}

// -----------------------------------------------------------------------------
test {
  testInter();
  testBasic();
  callfunc();
}
