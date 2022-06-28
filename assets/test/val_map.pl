fn test1() {
  assert::eq({}:length(), 0);
  assert::eq({}:set("a", "b"), {"a": "b"});
  assert::eq({"a":"b"}:get("a"), "b");
  assert::eq({}:tryGet("a", "b"), "b");
  assert::eq({"a":"b"}:length():to_string(), "1");
  assert::eq({"a":"b"}:del("a"):length(), 0);
}

fn testMapKey() {
  {
    let kv = {
      "a": 0,
      ["a" + "b"]: 1,
      "c": 2
    };

    assert::eq(kv.a, 0);
    assert::eq(kv.ab, 1);
    assert::eq(kv.c, 2);
    assert::yes(kv:has("a"));
    assert::no(kv:has("ccc"));
  }
}

fn testMapIter() {
  assert::eq(|{
    let i = 0;
    for let _, _ = {} {
      i++;
    }
    i;
  }, 0);
  assert::eq(|{
    let i = 0;
    for let _, _ = {'a': 1} {
      i++;
    }
    i;
  }, 1);
  assert::eq(|{
    let i = 0;
    for let _, _ = {'a': 1, 'b': 2} {
      i++;
    }
    i;
  }, 2);
}

fn testMapIndex() {
  assert::throw(
    fn() {
      let v = {}["a"];
    }
  );
  assert::eq(try ({}["a"]) else 100, 100);
  assert::eq(try ({}["b"]) else 200, 200);
  assert::eq({"a": 1}["a"], 1);
}

fn testBasic() {
  assert::eq(type({}), "map");
}

fn testNested() {
  assert::eq(
    {
      "a" : {
        "a" : {
          "a" : {
            "a" : {
              "a" : {
                "a" : {
                  "a" : {
                    "a" : {
                      "a" : {
                        "a" : 1
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }.a.a.a.a.a.a.a.a.a.a, 1);
}

test {
  test1();
  testMapKey();
  testMapIter();
  testMapIndex();
  testBasic();
  testNested();
}
