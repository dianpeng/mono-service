error => {
	dprint(phase, error);
	status => 503;
	body => "whatever shit: {{phase}} => {{error}}\n";
}

// sign policy, used for handling signing result
sign => {
	status => 200;
	header_set => ("x-body-sign-final-result", sign);
	header_set => ("x-body-sign-method", signMethod);

	// notes, we need to use signBody, since the original body has been
	// consumed up
	body => signBody;
};

// policy for handling verification result
// reject means the verification failed; pass means the verification
// passed
reject => {
	status => 404;
	header_set  => [
		("x-body-sign-result", sign),
		("x-body-sign-expect", signExpect)
	];

	// notes, we need to use signBody, since the original body has been
	// consumed up
	body => signBody;
};

pass => {
	status => 202;
	header_set => [
		("x-body-sign-result", sign),
		("x-body-sign-expect", signExpect)
	];
	body => signBody;
};

log => {
	dprint(logFormat);
}
