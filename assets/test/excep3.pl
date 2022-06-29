/**
 * exception handling nested failure
 */

fn test1() {
  assert::eq(
    try 
      try foo else bar
    else let _ 10,
    10);
}

test {
  test1();
}

