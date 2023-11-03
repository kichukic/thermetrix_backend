package tableconfig

// übergeordnete struct zur Verwaltung
type TableConfig struct {
	ConfigType string        `json:"config_type"`
	Config     SCTableConfig `json:"config"`
}
type TableConfigs []TableConfig

type SCTableConfig struct {
	TableHeaders        SCTableHeaders `json:"table_headers"`
	TableHeadersDisplay []string       `json:"table_headers_display"`
	TableActions        SCTableActions `json:"table_actions"`
}
type SCTableConfigs []SCTableConfig

type SCTableAction struct {
	Index string `json:"index,omitempty"`
	Label string `json:"label,omitempty"`
	Icon  string `json:"icon,omitempty"`
}
type SCTableActions []SCTableAction

type SCTableHeaderBasic struct {
	Title           string                 `json:"title,omitempty"`
	DisplayBy       string                 `json:"display_by,omitempty"`
	DisplayArrayBy  string                 `json:"display_array_by,omitempty"`
	ConcatWith      string                 `json:"concat_with,omitempty"`
	ConcatArrayWith string                 `json:"concat_array_with,omitempty"`
	Type            string                 `json:"type,omitempty"` //'string' | 'number' | 'date' | 'currency' | 'boolean'
	HtmlTemplate    string                 `json:"html_template,omitempty"`
	Variables       SCTableHeaderVariables `json:"variables,omitempty"`
	BooleanValues   string                 `json:"boolean_values,omitempty"`
}

type SCTableHeader struct {
	Index                   string                        `json:"index,omitempty"`
	Title                   string                        `json:"title,omitempty"`
	DisplayBy               string                        `json:"display_by,omitempty"`
	DisplayArrayBy          string                        `json:"display_array_by,omitempty"`
	ConcatWith              string                        `json:"concat_with,omitempty"`
	ConcatArrayWith         string                        `json:"concat_array_with,omitempty"`
	Type                    string                        `json:"type,omitempty"` //'string' | 'number' | 'date' | 'currency' | 'boolean'
	HtmlTemplate            string                        `json:"html_template,omitempty"`
	Variables               SCTableHeaderVariables        `json:"variables,omitempty"`
	BooleanValues           string                        `json:"boolean_values,omitempty"`
	SubtitleTitle           string                        `json:"subtitle_title,omitempty"`
	SubtitleDisplayBy       string                        `json:"subtitle_display_by,omitempty"`
	SubtitleDisplayArrayBy  string                        `json:"subtitle_display_array_by,omitempty"`
	SubtitleConcatWith      string                        `json:"subtitle_concat_with,omitempty"`
	SubtitleConcatArrayWith string                        `json:"subtitle_concat_array_with,omitempty"`
	SubtitleType            string                        `json:"subtitle_type,omitempty"`
	SubtitleHtmlTemplate    string                        `json:"subtitle_html_template,omitempty"`
	SubtitleVariables       SCTableHeaderVariables        `json:"subtitle_variables,omitempty"`
	SubtitleBooleanValues   string                        `json:"subtitle_boolean_values,omitempty"`
	Sticky                  bool                          `json:"sticky,omitempty"`
	StickyEnd               bool                          `json:"sticky_end,omitempty"`
	Img                     SCTableHeaderImg              `json:"img,omitempty"`
	Icon                    SCTableHeaderIcon             `json:"icon,omitempty"`
	TruncateAfter           int                           `json:"truncate_after,omitempty"`
	DateFormat              string                        `json:"date_format,omitempty"`
	CurrencyCodeDisplayBy   string                        `json:"currency_code_display_by,omitempty"`
	CurrencyCode            string                        `json:"currency_code,omitempty"`
	Align                   string                        `json:"align,omitempty"`
	Style                   string                        `json:"style,omitempty"`
	Styles                  string                        `json:"styles,omitempty"`        //  { style: string, conditions?: SCTableCondition[] }[]
	InlineStyles            string                        `json:"inline_styles,omitempty"` // { style: string, conditions?: SCTableCondition[], variables: SCTableHeaderVariable[] }[]
	Subtitle                *SCTableHeaderBasic           `json:"subtitle,omitempty"`
	Subtitles               []string                      `json:"subtitles,omitempty"`
	Filters                 SCTableFilters                `json:"filters,omitempty"`
	FilterCategories        SCTableFilterCategories       `json:"filter_categories,omitempty"`
	FilterIndexes           []string                      `json:"filter_indexes,omitempty"`
	FilterCategoryIndexes   SCTableHeaderFilterCategories `json:"filter_category_indexes,omitempty"`
	DisableSort             bool                          `json:"disable_sort,omitempty"`
}
type SCTableHeaders []SCTableHeader

type SCTableHeaderVariable struct {
	DisplayBy   string `json:"display_by,omitempty"`
	ConcatWith  string `json:"concat_with,omitempty"`
	VariableKey string `json:"variable_key,omitempty"`
}
type SCTableHeaderVariables []SCTableHeaderVariable

type SCTableHeaderImg struct {
	Url       string                 `json:"url,omitempty"`
	Style     string                 `json:"style,omitempty"` // 'square' | 'rounded'; //Style to display for img
	Variables SCTableHeaderVariables `json:"variables,omitempty"`
}
type SCTableHeaderImgs []SCTableHeaderImg

type SCTableHeaderIcon struct {
	Icon      string                 `json:"icon,omitempty"`
	MatIcon   string                 `json:"mat_icon,omitempty"`
	FontIcon  string                 `json:"font_icon,omitempty"`
	Classes   string                 `json:"classes,omitempty"`
	Variables SCTableHeaderVariables `fontIcon:"variables,omitempty"`
}
type SCTableHeaderIcons []SCTableHeaderIcon

type SCTableFilter struct {
	Index             string        `json:"index,omitempty"`
	Label             string        `json:"label,omitempty"`
	Icon              string        `json:"icon,omitempty"`
	Type              string        `json:"type,omitempty"`        // 'slider' | 'sliderrange' | 'select' | 'multiselect' | 'date' | 'toggle' | 'multitoggle' | 'checkbox' //Type of filter
	Data              string        `json:"data,omitempty"`        //  { value: T; label: string, icon?: string, matIcon?: string, fontIcon?: string }[]; //Values in case of sc-select/sc-autocomplete
	DataSource        interface{}   `json:"data_source,omitempty"` // Observable<{ value: T; label: string, icon?: string, matIcon?: string, fontIcon?: string }[] | any>; //Values in case of sc-select/sc-autocomplete
	FilterBy          string        `json:"filter_by,omitempty"`
	ValueBy           string        `json:"value_by,omitempty"`
	DisplayBy         string        `json:"display_by,omitempty"`
	DisplayByArray    string        `json:"display_by_array,omitempty"`
	selected          []interface{} `selected:"icon,omitempty"`
	Options           string        `json:"options,omitempty"` // { compareWith?: '<' | '>' | '<=' | '>=' | '==' };
	AdditionalOptions interface{}   `json:"additional_options,omitempty"`
	SearchControl     interface{}   `json:"search_control,omitempty"`
	IgnoreFilter      bool          `json:"ignore_filter,omitempty"`
}
type SCTableFilters []SCTableFilter

type SCTableFilterCategory struct {
	Index   string         `json:"index,omitempty"`
	Label   string         `json:"label,omitempty"`
	Filters SCTableFilters `json:"filters,omitempty"`
}
type SCTableFilterCategories []SCTableFilterCategory

type SCTableHeaderFilterCategory struct {
	Index         string   `json:"index,omitempty"`
	FilterIndexes []string `json:"filter_indexes,omitempty"`
}
type SCTableHeaderFilterCategories []SCTableHeaderFilterCategory

type SCTableCondition struct {
	CompareBy string      `json:"compare_by,omitempty"`
	Compare   string      `json:"compare,omitempty"` //  '==' | '<' | '>' | '<=' | '>=' | '!=';
	Value     interface{} `json:"value,omitempty"`
	Values    string      `json:"values,omitempty"` // { compare: '==' | '<' | '>' | '<=' | '>=' | '!='; value: T }[]; //Values chained with AND
}
type SCTableConditions []SCTableCondition
