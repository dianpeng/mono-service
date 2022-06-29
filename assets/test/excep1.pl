/* testing exception expression */
fn testBasic() {
  assert::eq(try [][1] else 10, 10);
  assert::eq(try v else 10, 10);
  assert::eq(try foo() else 10, 10);
}

fn testNested1() {
  assert::eq(
    try
      try
        try
          try v else 10
        else 11
      else 12
    else 13,
  10);

  assert::eq(
    try (
      try (
        try (
          try (
            try (
              try(
                (fn(){
                  let v = foo;
                })() ) else 10) else 11
          ) else 12
        ) else 13
      ) else 14
    ) else 15,
    10
  );
}

// nesting exception handling via different function scopes
fn t2_1() {
  return try t2_2() else 100;
}

fn t2_2() {
  t2_3();
}

fn t2_3() {
  t2_4();
}

fn t2_4() {
  t2_5();
}

fn t2_5() {
  let x = v; // error
}

fn testNested2() {
  assert::eq(t2_1(), 100);
}

// nesting exception handling via different function scopes
fn t3_1() {
  return try t3_2() else 100;
}

fn t3_2() {
  return t3_3();
}

fn t3_3() {
  return try t3_4() else 10;
}

fn t3_4() {
  t3_5();
}

fn t3_5() {
  let x = v; // error
}

fn testNested3() {
  assert::eq(t3_1(), 10);
}

// handling exception in the rule
fn t4_1() {
  t4_2();
}

fn t4_2() {
  t4_3();
}

fn t4_3() {
  t4_4();
}

test {
  testBasic();
  testNested1();
  testNested2();
  testNested3();
  assert::eq(try t4_1() else 10, 10);
}
