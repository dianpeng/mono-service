// capture from upper function lexical scope
fn gen1(bar) {
  let value = 10;
  return fn() {
    return value + bar;
  };
}

fn test1() {
  assert::eq(gen1(10)(), 20);
}

// modify captured value with heap semantic
fn test2() {
  let value = {};
  let local = fn() {
    value:set("a", "b");
  };
  local();
  assert::eq(value.a, "b");
}

test {
  test1();
  test2();

  // capture from the rule
  {
    let v1 = 10;
    {
      let v2 = [1];
      {
        let v3 = {"a":1, "b":2};

        slot => fn() {
          let vv1 = v1;
          let vv2 = v2;
          let vv3 = v3;
          return v1 + vv2:length() + vv3:length();
        };
      }
    }
  }
  assert::eq(slot(), 13);
}
