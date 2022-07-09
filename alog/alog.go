package alog

type Format struct {
	Raw string
	bc  program
}

type Log struct {
	Format   *Format
	Appendix []string
}

func NewLog(
	f *Format,
) Log {
	return Log{
		Format: f,
	}
}

func CompileFormat(input string) (*Format, error) {
	p := formatParser{}
	if err := p.parse(input); err != nil {
		return nil, err
	}
	return &Format{
		Raw: input,
		bc:  p.prog,
	}, nil
}

func (l *Log) ToText(
	p Provider,
	emptyPlaceholder string,
	delimiter string,
) string {
	buf := toText(
		l.Format.bc,
		p,
		emptyPlaceholder,
		delimiter,
	)

	for _, a := range l.Appendix {
		buf.WriteString(a)
		buf.WriteString(delimiter)
	}

	return buf.String()
}
