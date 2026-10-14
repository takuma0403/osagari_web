package domain

// Department は学部・学科（AUTH-4 の表の1行）。
type Department struct {
	// Code は学籍番号の3〜5桁目の学部・学科コード
	Code string
	// Name は表示名（例「理工学部 理工学科」）
	Name string
	// Years は修業年限。有効期限の計算に使う（AUTH-3）
	Years int
}

// defaultYears は表にない学部・学科コードの修業年限（AUTH-4「表にないコードの扱い」）。
const defaultYears = 4

// departments は AUTH-4 の表の写し。表が更新されたらここだけを直す。
// 大学院の研究科コードと改組前のコードは未確認のため載せない（表にないコードは不明として扱う。docs/product/open-questions.md）。
var departments = map[string]Department{
	"101": {"101", "教育学部 学校教育課程", 4},
	"121": {"121", "芸術地域デザイン学部 芸術地域デザイン学科", 4},
	"141": {"141", "経済学部 経済学科", 4},
	"142": {"142", "経済学部 経営学科", 4},
	"143": {"143", "経済学部 経済法学科", 4},
	"211": {"211", "医学部 医学科", 6},
	"212": {"212", "医学部 看護学科", 4},
	"238": {"238", "理工学部 理工学科", 4},
	"254": {"254", "農学部 生物資源科学科", 4},
	"271": {"271", "コスメティックサイエンス学環", 4},
}

// LookupDepartment は学部・学科コードに対応する学部・学科を返す。表にないコードなら false。
func LookupDepartment(code string) (Department, bool) {
	d, ok := departments[code]
	return d, ok
}
