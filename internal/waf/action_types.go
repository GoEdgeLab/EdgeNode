package waf

import "reflect"

type ActionString = string

const (
	ActionLog              ActionString = "log"       // allow and log
	ActionBlock            ActionString = "block"     // block
	ActionCaptcha          ActionString = "captcha"   // block and show captcha
	ActionJavascriptCookie ActionString = "js_cookie" // js cookie
	ActionNotify           ActionString = "notify"    // 告警
	ActionGet302           ActionString = "get_302"   // 针对GET的302重定向认证
	ActionPost307          ActionString = "post_307"  // 针对POST的307重定向认证
	ActionRecordIP         ActionString = "record_ip" // 记录IP
	ActionTag              ActionString = "tag"       // 标签
	ActionPage             ActionString = "page"      // 显示网页
	ActionRedirect         ActionString = "redirect"  // 跳转
	ActionAllow            ActionString = "allow"     // allow
	ActionGoGroup          ActionString = "go_group"  // go to next rule group
	ActionGoSet            ActionString = "go_set"    // go to next rule set
)

var AllActions = []*ActionDefinition{
	{
		Name:     "阻止",
		Code:     ActionBlock,
		Instance: new(BlockAction),
		Type:     reflect.TypeOf(new(BlockAction)).Elem(),
	},
	{
		Name:     "允许通过",
		Code:     ActionAllow,
		Instance: new(AllowAction),
		Type:     reflect.TypeOf(new(AllowAction)).Elem(),
	},
	{
		Name:     "允许并记录日志",
		Code:     ActionLog,
		Instance: new(LogAction),
		Type:     reflect.TypeOf(new(LogAction)).Elem(),
	},
	{
		Name:     "Captcha验证码",
		Code:     ActionCaptcha,
		Instance: new(CaptchaAction),
		Type:     reflect.TypeOf(new(CaptchaAction)).Elem(),
	},
	{
		Name:     "JS Cookie验证",
		Code:     ActionJavascriptCookie,
		Instance: new(JSCookieAction),
		Type:     reflect.TypeOf(new(JSCookieAction)).Elem(),
	},
	{
		Name:     "告警",
		Code:     ActionNotify,
		Instance: new(NotifyAction),
		Type:     reflect.TypeOf(new(NotifyAction)).Elem(),
	},
	{
		Name:     "GET 302",
		Code:     ActionGet302,
		Instance: new(Get302Action),
		Type:     reflect.TypeOf(new(Get302Action)).Elem(),
	},
	{
		Name:     "POST 307",
		Code:     ActionPost307,
		Instance: new(Post307Action),
		Type:     reflect.TypeOf(new(Post307Action)).Elem(),
	},
	{
		Name:     "记录IP",
		Code:     ActionRecordIP,
		Instance: new(RecordIPAction),
		Type:     reflect.TypeOf(new(RecordIPAction)).Elem(),
	},
	{
		Name:     "标签",
		Code:     ActionTag,
		Instance: new(TagAction),
		Type:     reflect.TypeOf(new(TagAction)).Elem(),
	},
	{
		Name:     "显示页面",
		Code:     ActionPage,
		Instance: new(PageAction),
		Type:     reflect.TypeOf(new(PageAction)).Elem(),
	},
	{
		Name:     "跳转",
		Code:     ActionRedirect,
		Instance: new(RedirectAction),
		Type:     reflect.TypeOf(new(RedirectAction)).Elem(),
	},
	{
		Name:     "跳到下一个规则分组",
		Code:     ActionGoGroup,
		Instance: new(GoGroupAction),
		Type:     reflect.TypeOf(new(GoGroupAction)).Elem(),
	},
	{
		Name:     "跳到下一个规则集",
		Code:     ActionGoSet,
		Instance: new(GoSetAction),
		Type:     reflect.TypeOf(new(GoSetAction)).Elem(),
	},
}
