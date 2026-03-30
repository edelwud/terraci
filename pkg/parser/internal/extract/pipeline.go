package extract

type extractorStep func(*Context)

func RunDefault(ctx *Context) {
	newSession(ctx).Run()
}
