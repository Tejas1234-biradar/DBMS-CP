package utils

const (
	INDEXER_QUEUE_KEY = "pages_queue"
	SIGNAL_QUEUE_KEY  = "signal_queue"
	RESUME_CRAWL      = "RESUME_CRAWL"

	NORMALIZED_URL_PREFIX = "normalized_url"
	URL_METADATA_PREFIX   = "url_metadata"
	PAGE_PREFIX           = "page_data"
	WORD_PREFIX           = "word"
	BACKLINKS_PREFIX      = "backlinks"
	OUTLINKS_PREFIX       = "outlinks"

	// # Maximum words to index
	MAX_INDEX_WORDS = 1000
	// # Common Top-Level Domains (TLDs) this is clearly AI generated
)

var FileTypes = map[string]struct{}{
	"jpg": {}, "jpeg": {}, "png": {}, "gif": {}, "webp": {},
	"pdf": {}, "doc": {}, "docx": {}, "xls": {}, "xlsx": {},
	"mp4": {}, "mp3": {}, "wav": {}, "avi": {}, "mov": {},
	"zip": {}, "tar": {}, "gz": {}, "rar": {},
	"css": {}, "js": {}, "xml": {}, "json": {}, "csv": {},
	"svg": {}, "ico": {}, "ttf": {}, "woff": {}, "woff2": {},
}

var PopularDomains = map[string]struct{}{
	// Generic TLDs
	"com": {}, "org": {}, "net": {}, "edu": {}, "gov": {},
	"mil": {}, "int": {}, "biz": {}, "info": {}, "name": {},
	"pro": {}, "xyz": {}, "online": {}, "site": {}, "shop": {},
	"store": {}, "blog": {}, "news": {}, "media": {}, "art": {},
	"film": {}, "game": {}, "games": {}, "tech": {}, "app": {},
	"dev": {}, "ai": {}, "cloud": {}, "io": {}, "co": {},
	"me": {}, "tv": {}, "ly": {}, "to": {}, "fm": {},
	"wiki": {}, "help": {},
	// Country Code TLDs
	"us": {}, "uk": {}, "ca": {}, "au": {}, "de": {},
	"fr": {}, "jp": {}, "cn": {}, "ru": {}, "br": {},
	"in": {}, "cl": {}, "mx": {}, "es": {}, "it": {},
	"nl": {}, "se": {}, "no": {}, "fi": {}, "dk": {},
	"pl": {}, "be": {}, "ch": {}, "at": {}, "nz": {},
	"za": {}, "sg": {}, "hk": {}, "kr": {}, "id": {},
	"my": {}, "ph": {}, "th": {}, "vn": {}, "il": {},
	"sa": {}, "ae": {}, "tr": {}, "eg": {}, "ar": {},
	"pe": {}, "ve": {}, "pk": {}, "ng": {}, "ke": {},
	"tz": {}, "ro": {},
	// Language Codes
	"en": {}, "pt": {}, "zh": {}, "ja": {}, "ko": {},
	"ms": {}, "hi": {}, "bn": {}, "ur": {}, "vi": {},
	// Popular Domains / Brands
	"google": {}, "facebook": {}, "instagram": {}, "twitter": {}, "tiktok": {},
	"linkedin": {}, "youtube": {}, "reddit": {}, "wikipedia": {}, "yahoo": {},
	"bing": {}, "microsoft": {}, "apple": {}, "amazon": {}, "ebay": {},
	"netflix": {}, "hulu": {}, "spotify": {}, "pinterest": {}, "snapchat": {},
	"discord": {}, "steam": {}, "github": {}, "gitlab": {}, "bitbucket": {},
	"twitch": {}, "paypal": {}, "stripe": {}, "wordpress": {}, "tumblr": {},
	"medium": {}, "quora": {}, "stackoverflow": {}, "dropbox": {}, "icloud": {},
	"adobe": {}, "salesforce": {}, "slack": {}, "zoom": {}, "airbnb": {},
	"uber": {}, "lyft": {}, "doordash": {}, "tesla": {}, "openai": {},
	"nvidia": {}, "amd": {}, "intel": {}, "samsung": {}, "huawei": {},
	"xiaomi": {}, "sony": {}, "bbc": {}, "cnn": {}, "nytimes": {},
	"forbes": {}, "bloomberg": {}, "wsj": {}, "reuters": {},
}
