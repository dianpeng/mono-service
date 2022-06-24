/**
 * testing basic function, anoymous function and upvalue closure
 **/
global {
  a_const = "from const";
}

session {
  a_func = fn(a, b, c) {
    return a + b + c;
  };
}


fn foo1() {
  return "hello world";
}

fn foo2(a) {
  return a:to_upper();
}

fn foo3(a, b) {
  return a + b;
}

fn testBasic() {
  assert::eq(foo1(), "hello world");
  assert::eq(foo2("hello"), "HELLO");
  assert::eq(foo3(1, 2), 3);
  assert::eq(a_func(1, 2, 3), 6);
  assert::eq(session::a_func(1, 2, 3), 6);
  assert::eq(global::a_const, "from const");
  assert::eq(a_const, "from const");

  const local_f0 = fn() {
    return "local_f0";
  };

  let local_f1 = fn() {
    return "local_f1";
  };

  assert::eq(local_f0(), "local_f0");
  assert::eq(local_f1(), "local_f1");
}

// Testing upvalue -------------------------------------------------------------
fn uv1() {
  let u1 = 10;
  return fn() {
    return fn() {
      return u1;
    };
  };
}

fn uv2(u0) {
  let u1 = 10;
  let u2 = 20;
  let u3 = 30;
  let u4 = 40;
  return fn() {
    return fn() {
      return fn() {
        return fn() {
          return u0 + u1 + u2 + u3 + u4;
        };
      };
    };
  };
}

fn testUpvalue() {
  assert::eq(uv1()()(), 10);
  assert::eq(uv2(0)()()()(), 10+20+30+40);
}

// ----------------------------------- ENTRY -----------------------------------
test {
  testBasic();
  testUpvalue();
}
