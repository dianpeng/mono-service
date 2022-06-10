package module

// Concate services, ie generate a merge/concate of multiple http upstream
// content and send them as downstream response. This module allows highly
// flexible customization via policy engine.

// Each request will be splitted into 4 different phases. And the go code will
// drive each phase enter and leave

// 1. url
//    In this phase, the policy engine's is responsible to generate a output
//    via action "url" which contains a list of strings that indicates the
//    URL a request must be made against to

// 2. request
//    In this phase, for each url in the lists, the request event will be
//    emitted and user is capable of fine grained customize the request by
//    setting its request's method, header, body, most importantly, a pass_status
//    action allow user to setup a failure condition when the request cannot be
//    made to downstream

// 3. error
//    In this phase, the event handler will only be triggered when certain
//    http requests failed, either because of it cannot pass pass_status action
//    or network error etc ...
//    User is able to overwrite the downstream body or reject the whole request

// 4. downstream
//    In this phase, the user event is able to perform last modification of the
//    http response, ie modifying its response header, its response status code.
//    Notes the response code's default value has already been set when enter
//    into this phase
