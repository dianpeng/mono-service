
# PL (Policy Language)

## Introduction


PL is specifically designed for web user. For web user, typically will embed a simple language engine
, particularly Lua in many cases, for extending its functionality. PL is designed from scratch to be a
modern script language specifically for web/http use cases. It has many considerations to improve user
experience inside of web/http server environment. It is a reflection of myself with many years of web/http
server implementation or related industry experience.


## Highlights

* Rich Language Constructs
  1. Regex literal
  2. Template (go, markdown, ...) literal
  3. Exception Handling
  4. Json interpobility
  5. Lots of builtin modules, easy to extend
  6. String interpolation, const, immutablility, closure upvalue captured etc ...

* Designed For Web/Http Server
  1. Event driven
  2. Integrated with Go reflection to enable easier function binding
  3. Session variables

* Reasonably Good Performance
  1. One pass code generation
  2. Bytecode style VM with compact memory footprint

## Language


### Basic Language Features


#### Type

PL features a highly flexible type system with following category

* Primitive
  1. Int
  2. Real (Float)
  3. Null
  4. Bool
  5. String

* Compound
  1. List
  2. Map
  3. Regexp
  4. Pair (a struct that has First and Second field)

* Other
  1. Iterator (used for iterator loop)
  2. Closure (any callable object, ie script defined function or user defined function)
  3. Usr (Go/Native extension)

#### Constructs

PL supports 2 types of execution unit

1. Function

Like most programming language, function is a union of statements. In PL, function is first order element. ie,
function is a value and can be passed around. Additionally, function supports implicit value capture to form
closure. Accessing variable which is not available inside of function's own lexical scope but up to its parental
nested scope is allowed.

```

fn foo() {
  let v = 100;
  return fn() {
	return fn() {
    	return v; // this is allowed, v will be captured
    };
  };
};

```


2. Rule

PL is a policy language which features a event driven execution. Instead of having function, the program typically starts to execute in some of the rules. A rule is just like function instead it must be defined
at the top level and it is not a value semantic. Additionally, rule supports special matching grammar. The
VM will iterate each rule and pick up one to execute. Each rule context has an implicit variable  $and this $
represents current event name and is used to pick up specific rule.


```

// simple rule definition, this rule matches when the event is "a"
rule a {
}

// you can define rule's name with quoted string as well
rule "b" {
}

// or quoted with square
rule [c] {
}

// or ignore rule keyword entirely
cc {
}

// you can define a customize matching rule.
// If so, you should explicitly write the matching rule. Notes, if you do not
// write match explicitly, the compiler will generate code for matching, same as
// matching current event name with you defined rule name.

rule dd

when $ == "dd" && // matching the event name with dd, since we write it explicitly
     request.method == "GET" // must be a GET method to trigger in our web server

{

}

// inside of the rule body, user is allowed use special grammar called action. Action is used to
// return value from rule. Instead of returning from rule and terminate execution, action will
// not terminate but just yield the value out externally. Notes, user is not allowed write return
// inside of the rule body, which is a syntax error.

rule xx {
  status => 200; // action, yield 200 to status action
  body => "Hello World"; // action, yield string Hello World to body action

  foo();  // execution will still reach here and will not interrupt because of action
}

```


#### Variable and Scoping Rules

PL supports 4 types of variable/symbol

* Dynamic

Dynamic Variable is unknown variable exposed into the PL environment by embedding environment. It
requires dynamic lookup during code execution and my result in error if environment does not expose
it back to environment

* Global

Global variable is variable defined inside of the global scope, which is immutable and setup initially.
Global scope must be top level scope of the source and MUST be the first scope otherwise it is a syntax
error.


```

global {
 const1 = http::do("GET", "http://www.example.com"); // will just be initialized once
}

fn foo() {
}

// the following const scope is not allowed and will not compile
global {
...
}

```

* Session

Session variable is variable defined inside of the session scope. In our web server context, the session will
be re-evaluated for each HTTP transaction. Notes it is up to the embedder to express its semantics and the PL
engine has no knowledge of how the session variable is been used. Notes, Session scope must be at the top of
the source code *just after* the const scope, if there's a const scope. Otherwise it is a syntax error.

```

session {
  session = http::do("GET", "http://www.example.com"); // will be variable and will be re-evaluated each session
}

fn foo() {
}

// the following const scope is not allowed and will not compile
session {
  ...
}

```

* Local

Local is just local variable defined anywhere inside of the rule or function body. It can be defined as const or variable via different keyword

```

rule a {
  let a_var = 100;
  a_var = 200;

  const a_const = 10;

  // this is a compile error, since a_const is const local and cannot be modified
  a_const = 20;
}

// same name local variable can be nested inside of different scopes

rule a {
  let xx = 1000;
  {
	let xx = 100;
    {
      let xx = 10;
      assert::eq(xx, 10);
    }
    assert::eq(xx, 100);
  }
  assert::eq(xx, 1000);
}

```


#### Expression

* Basic Expression

```
  let a = 1+2*3;
  let b = 1 <= 3+4;
  let c = true != false;
  let d = "abc" + 'def'; // string literal can be quoted with single or double quotes
  let e = 2.0 * 3;
  let g = [1, 2, 3, true, null, "Hello World"]; // list literal
  let h = {
     'a' : "Hell",
     "b" : " ",
     "c" : "World"
  };  // map literal, ie a look up dictionary

  let j = fn(a, b, c) {
	return a+b+c;
  }; // anonymous function binding, we will have detail talk about how function/closure works

  // this is a pair expression, which result in a internal type *pair*
  let pair = ("a", "b");

  // this is a sub expression
  let subexpression = (1+2);

  // a super long heredoc
  let long_string = ```EOF
	This is a multiple
    line string
  EOF```;

  // Additionally, to allow easy formating of string, PL features a string interpolation syntax
  // anything inside of the string literal which sits between {{...}} will be treated as valid
  // PL expression for evaluation. Notes the string interpolation will be off inside of the
  // template rendering context
  let fancy_string = "this is a variable {{e}} and a functino call {{foo(1, 2, 3)}}";

  // We also support regex literal, regex has its special operator ~ and !~ for matching and not
  // match semantic

  let regex = r"This is a regex string";
  let matched = "A string" ~ regex; // returns true or false

  // pipe syntax is also support for better expressive, pipe is essentially just
  // a function call
  // a | b is same as b(a)
  // a | b(2, 3) is same as b(a, 2, 3)

  let pipe1 = "string" | str::to_upper;
  let mul10 = fn(a, b) {
    return (a + b) * 10;
  };
  let pipe2 = 1 | mul10(2);

  // additionally, we also support method call. A method call is a specialized
  // call that dispatch directly to an object that support method call.

  let x = "lower":to_upper(); // use : to indicate calling method to_upper on string

  // be careful the difference between method call and dot call.

  let obj = {
  };

  obj.a = fn() {
    return "this is a closure";
  };

  let aa = obj.a(); // this is a call of field a inside of map *obj*

  let bb = obj:length(); // this is a call of length member function on map *obj*

```

* Ternary Expression

Instead of using C style ternary, we use python way, ie via if else keyword to make the ternary
much more clean. Additionally, our if else does support scoping, so user can put multiple statements
inside of the body of if else scope and the last statement's value will be used as whole if elif else
branch's value output. Pretty much like rust.

```

// basic python ternary, compact way ternary

let v1 = "true" if false == false else "false";

// if block style expression. Notes, inside of the if branch, only following statement is
// allowed to use:
//
// 1) let statement
// 2) const statement
// 3) for loop
// 4) expression statement, like expression or function call etc ...
//
// other control flow like return is not allowed

let v2 = if false == false {
  foo();
  bar();
  voo();
  100; // this expression will be used as v2's if branch value
} else {
  200; // this expression will be used as v2's else branch value
};

// the following if expression does not contain a elif or else branch, so the compiler will
// generate a null when the branch if is not hit
let v3 = if false = true {
  100;
};

// this will generate an error since the following if body DOES NOT generate any value.
// The first let statement does not generate any value
let v3 = if false == false {
  let x = 100;
  for { ... }; // for loop does not generate any value
};


```

* Expression Exception Handling

Used to handle expression level error

```

let g = 10;

// the following expression uses try [expression] else [value] style to handle exception
// in this case, symbol g is a local symbol has value 10 which cannot be invoked as function
// but with following expression, local variable xx will have value 100 because of the
// exception handling.

let xx = try g() else 100;

```

* Template Expression

Template expression allow user to directly embed template into language or from external file. The template
compliation happened during the PL language compilation. Each template's compiled assets will be cached once
after the PL script been compiled for performance.

```

let myGoTemplate = template "go", {"hello" : "world"}, ```EOF
  This is from template {{.hello}} world
EOF```;

```

* Qualified Variable Lookup

You can force the compiler to resolve certain symbol with certain type. Typically, if variable a has name
collision, for example there's a session variable called "a" and also a local variable called "a". Then the
local variable will be bounded to any identifier reference with "a" inside of expression. But you can use
scope qualifier to force compiler to lookup certain types of value.

```

global {
  a = 10;
}

session {
  a = 10;
}

rule xx {

  let a = 10;

  let v1 = session::a; // force lookup symbol a as session variable
  let v2 = global::a;   // force lookup symbol a as const variable
  let v3 = dyanmic::a; // force lookup symbol a as dynamic variable
  let v4 = a;          // no qualifier, default to basic lookup rule, and local variable takes precedence
}

```

* Anonymous Function and Closure

PL supports first order function and lexical scope variable up value captured. The primitive type value is
captured by value and any none primitive type is captured by reference. User can capture any value from the
closest enclosed function lexical scope to top most.


```

fn nested_func() {
  let a = 1; // a upvalue value

  return fn() {
    return fn() {
      return fn() {
        return a; // you can capture the top most nested function's upvalue.
      };
    };
  };
}

rule xx {
  assert::eq(nested_func()()()(), 1);
}

// aslo you can also capture value inside of the rule body

rule yy {
  let value = "this is a rule";

  assert::eq(
  	(fn() {
      return value;
    })(), "this is a rule");
}

// function can be passed aronud as value obviously

```


#### Control Flow


#### If

Like most language, PL does have a branch statement. The only different is PL's else if keyword is written as elif :).
Notes If statement and If expression is different. If statement is really a statement and it does not require the last
statement to be an expression. If expression grammar is only activated under expression context by parser, and the if expression requires the last statement inside of each if block to be an expression.

```

fn foo(flag) {
  if flag == 100 {
    return 100;
  } else {
    return "no";
  }
}

fn bar(flag) {
  if flag == 1 {
    return "A";
  } elif flag == 2 {
    return "B";
  } elif flag == 3 {
    return "C";
  } else {
    return "D";
  }
}

voodoo {
  if request.method == "GET" {
    status => 202;
  } elif request.method == "POST" {
    stauts => 400;
  }
}

```


### Loop

PL features just one loop keyword, for. And it supports 4 types of loop semantic as following :

1. Forever loop


```

for {
  let x = "I am Crazy!";
}

// never reach here. DO NOT DO THIS IN PRODUCTION

```


2. Trip count loop

Allowing user to setup initial statement, condition and step statement for loop to progress.


```

for let i = 1; i < 100; i++ {
}

// or if the induction variable has been defined and you want to reuse
let j = 0;
for j = 1; j < 100; j++ {
}

// user is allowed omit all there component of trip count loop, but semicolon must be left

for ; j < 1000; j += 2 {}

for ; ; j+= 3 {}

// this is essentially just a forever loop
for ;; {}

```

3. Iterator loop

Allowing user to iterate with certain types of object. User can extend type system to provide extension to
iterator.

```

// iterate through the list
for let index, value = [1, 2, 3, 4] {
}

// iterate through the map
for let key, value = {"a": 1, "b": 2} {
}


// notes iterator loop MUST starts with let keyword and must have exactly 2 elements to represent
// key/index and value for each iterator
// this is a syntax error since the compiler will try to parse it as a trip count loop
for let key = [1, 2, 3, 4] {
}

```

4. Condition loop

Still user can just set up a condition, like the C while loop.

```

// the loop will tick until the condition does not met

for a != 100 {
}

```


Loop control statement is allowed inside of the loop as other language. Notes we do not have goto or
label break statement as some language.


```

for {
  if xxx {
    break
  } else {
    continue
  }
}

```


### Call and Return

Inside of the rule or function, user is allowed to call other function. Inside of the function body, user can use
return to return value back and terminate the execution. Notes each function can return just one element for now.


```

fn foo() {
  return 1+2;
}

fn bar() {
  return foo();
}

rule xxx {
  bar();
  // return is not allowed in rule body
  return 10; // error
}

```

### Try

Similar as try expression, we do allow try statement as well. User can group multiple statement together and uses try and else block to handle code's exception/error. Notes, only runtime error can be handled, any compilation error will not be
handled as exception.


```

fn foo() {
  let v;

  try {
    v = dynamic_varaible; // suppose dynamic variable not existed
  } else {
    v = 100;
  }

  assert::eq(v, 100);
}

// notes session, global, local variable symbol resolution is during parsing/compilation phase which is
// a syntax error but not runtime error. The following code will not compile

fn not_valid() {
  let v = 10;
  try {
    v = session::a; // session variable a is not existed and it is been decided during compilation
  } else {
    v = 100;
  }
}

```
