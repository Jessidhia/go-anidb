package titles_test

import (
	"fmt"
	"github.com/Kovensky/go-anidb/titles"
	"os"
	"testing"
)

var db = &titles.TitlesDatabase{}

func init() {
	if fh, err := os.Open("anime-titles.dat.gz"); err == nil {
		db.LoadDB(fh)
	} else if fh, err = os.Open("anime-titles.dat"); err == nil {
		db.LoadDB(fh)
	} else {
		panic(err)
	}
}

type TestVector struct {
	Input string
	Limit int
	AIDs  []int
}

func TestFuzzySearch(T *testing.T) {
	// Each vector goes one step deeper in the fuzzy search stack
	vec := []TestVector{
		// no match
		TestVector{Input: "\x00", Limit: -1, AIDs: []int{}},
		// exact
		TestVector{Input: "SAC2", Limit: 1, AIDs: []int{1176}},
		// exact, but in hungarian!
		TestVector{Input: "Varázslatos álmok", Limit: -1, AIDs: []int{235}},
		// prefix words
		TestVector{Input: "Varázslatos", Limit: 3, AIDs: []int{235, 2152, 2538}},
		// suffix words
		TestVector{Input: "A rózsa ígérete", Limit: -1, AIDs: []int{2152}},
		// infix words
		TestVector{Input: "Stand Alone", Limit: 1, AIDs: []int{247}},
		// prefix
		TestVector{Input: "Ghost in t", Limit: 1, AIDs: []int{61}},
		// suffix
		TestVector{Input: "flowne", Limit: 1, AIDs: []int{184}},
		// words, first word first in name
		TestVector{Input: "Kumo Mukou", Limit: -1, AIDs: []int{469}},
		// words, last word last in name
		TestVector{Input: "A titka", Limit: 1, AIDs: []int{303}},
		// words, infix but not contiguous
		TestVector{Input: "Kidoutai 2nd", Limit: 1, AIDs: []int{1176}},
		// strings, first string first in name
		TestVector{Input: "Kouka Kidou", Limit: 1, AIDs: []int{61}},
		// strings, last string last in name
		TestVector{Input: "app Princess", Limit: 1, AIDs: []int{640}},
		// strings, anywhere in this order
		TestVector{Input: "ouka douta", Limit: 2, AIDs: []int{61, 247}},
		// match everything
		TestVector{Input: "", Limit: 1, AIDs: []int{1}},
	}

	for i, v := range vec {
		res := db.FuzzySearch(v.Input).ResultsByAID()
		if v.Limit > 0 && len(res) > v.Limit {
			res = res[:v.Limit]
		}

		wrong := false
		if len(v.AIDs) != len(res) {
			wrong = true
		} else {
			for j, r := range res {
				if v.AIDs[j] != r.AID {
					wrong = true
				}
			}
		}

		if wrong {
			list := make([]string, 0, len(res))
			for _, r := range res {
				list = append(list, fmt.Sprintf("%d (%s)", r.AID, r.PrimaryTitle))
			}
			T.Errorf("Vector #%d: Expected AID list %v, got AID list %v", i+1, v.AIDs, list)
		}
	}
}

func TestFuzzySearchFold(T *testing.T) {
	// Same vector as the previous one, but with disturbed word cases
	vec := []TestVector{
		// exact
		TestVector{Input: "sac2", Limit: 1, AIDs: []int{1176}},
		// exact, but in hungarian!
		TestVector{Input: "VarÁzslatos Álmok", Limit: -1, AIDs: []int{235}},
		// prefix words
		TestVector{Input: "varázslatos", Limit: 3, AIDs: []int{235, 2152, 2538}},
		// suffix words
		TestVector{Input: "a rÓzsa ígérete", Limit: -1, AIDs: []int{2152}},
		// infix words
		TestVector{Input: "Stand Alone", Limit: 1, AIDs: []int{247}},
		// prefix
		TestVector{Input: "ghost in t", Limit: 1, AIDs: []int{61}},
		// suffix
		TestVector{Input: "FlownE", Limit: 1, AIDs: []int{184}},
		// words, first word first in name
		TestVector{Input: "kumo mukou", Limit: -1, AIDs: []int{469}},
		// words, last word last in name
		TestVector{Input: "a titka", Limit: -1, AIDs: []int{303}},
		// words, infix but not contiguous
		TestVector{Input: "kidoutai 2nd", Limit: 1, AIDs: []int{1176}},
		// strings, first string first in name
		TestVector{Input: "Kouka kidou", Limit: 1, AIDs: []int{61}},
		// strings, last string last in name
		TestVector{Input: "app princess", Limit: 1, AIDs: []int{640}},
		// strings, anywhere in this order
		TestVector{Input: "Ouka Douta", Limit: 2, AIDs: []int{61, 247}},
		// no match
		TestVector{Input: "\x00", Limit: -1, AIDs: []int{}},
	}

	for i, v := range vec {
		res := db.FuzzySearchFold(v.Input).ResultsByAID()
		if v.Limit > 0 && len(res) > v.Limit {
			res = res[:v.Limit]
		}

		wrong := false
		if len(v.AIDs) != len(res) {
			wrong = true
		} else {
			for j, r := range res {
				if v.AIDs[j] != r.AID {
					wrong = true
				}
			}
		}

		if wrong {
			list := make([]string, 0, len(res))
			for _, r := range res {
				list = append(list, fmt.Sprintf("%d (%s)", r.AID, r.PrimaryTitle))
			}
			T.Errorf("Vector #%d: Expected AID list %v, got AID list %v", i+1, v.AIDs, list)
		}
	}
}

// exact match of primary title
func BenchmarkFuzzySearch_bestCase(B *testing.B) {
	// grep '|1|' anime-titles.dat | cut -d'|' -f4 | sort -R | sed 's/\(.*\)/"\1",/' | \
	//    head -n 30
	vec := []string{
		"Shin Tennis no Ouji-sama", "Shimai Ningyou", "Aniyome",
		"Dragon Ball Z: Kyokugen Battle!! Sandai Super Saiyajin", "Uchuu Kuubo Blue Noah",
		"Hotaru no Haka", "First Kiss Story: Kiss Kara Hajimaru Monogatari", "Seikai no Senki III",
		"Ikkitousen: Xtreme Xecutor", "Houkago Ren`ai Club: Koi no Etude",
		"DNA2: Dokoka de Nakushita Aitsu no Aitsu (1995)", "Bamboo Blade", "Accelerando",
		"Soukyuu no Fafner: Dead Aggressor", "Eiga Futari wa Precure Max Heart",
		"Kyoufu no Kyou-chan", "Shin Taketori Monogatari: 1000-nen Joou", "Fresh Precure!",
		"Grope: Yami no Naka no Kotori-tachi", "Seitokai Yakuindomo", "Chikyuu Shoujo Arjuna",
		"Choukou Tenshi Escalayer", "Dragon Ball Kai", "Dragon League", "Hatsukoi Limited",
		"Sexfriend", "Ao no Exorcist", "Futatsu no Spica", "Adesugata Mahou no Sannin Musume",
		"Yawara! A Fashionable Judo Girl",
	}

	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		db.FuzzySearch(vec[i%len(vec)])
	}
}

// // exact match of x-jat, en or ja non-primary title
// func BenchmarkFuzzySearch_secondBestCase(B *testing.B) {
// 	// grep -E '\|3\|(x-jat|en|ja)\|' anime-titles.dat | cut -d'|' -f4 | sort -R | \
// 	//    sed 's/\(.*\)/"\1",/' | head -n 30
// 	vec := []string{
// 		"yosusora", "heartcatch", "chuunibyou", "Stringendo", "おれいも", "yamato 2199",
// 		"mai otome zwei", "cg r1", "harem", "Dorvack", "Natsume 1", "SMJA", "SM", "J2",
// 		"amstv2", "BJ Movie (2005)", "munto2", "nyc", "MT", "DBZ Movie 2",
// 		"Zatch Bell Movie 2", "Armitage", "J0ker", "CH", "sugar", "vga", "Nadesico",
// 		"dgc nyo", "setv", "D.g", "マジプリ", "myyour", "Haruhi 2009", "bantorra", "yamato2",
// 		"bakuhan", "vk2", "BBB", "5-2", "GSD SE III", "akasaka", "GS SE II", "F3", "おれつば",
// 		"sencolle", "wellber", "SailorMoon", "ay", "HCPC", "kxstv", "Shana III",
// 	}

// 	B.ResetTimer()
// 	for i := 0; i < B.N; i++ {
// 		db.FuzzySearch(vec[i%len(vec)])
// 	}
// }

// // exact match of non-primary title in any other language
// func BenchmarkFuzzySearch_thirdBestCase(B *testing.B) {
// 	// grep '|2|' anime-titles.dat | grep -Ev '(x-jat|en|ja)' | cut -d'|' -f4 | \
// 	//    sort -R | sed 's/\(.*\)/"\1",/' | head -n 30
// 	vec := []string{
// 		"Зірка☆Щастя", "La ilusión de triunfar", "La scomparsa di Haruhi Suzumiya",
// 		"Код Геас: Бунтът на Люлюш 2", "我的女神 剧场版", "Lamu - Un rêve sans fin",
// 		"Lupin III: La cospirazione dei Fuma", "Адовая Девочка дубль 2", "夏娃的时间",
// 		"Дівчинка, що стрибала крізь всесвіт", "Мій сусід Тоторо", "机巧魔神",
// 		"City Hunter - Flash spécial !? La mort de Ryo Saeba", "Ateştopu", "مسدس×سيف",
// 		"Gli amici animali", "沉默的未知", "忧伤大人二之宫", "Пита-Тен", "Глава-гора", "高校龍中龍",
// 		"Яблочное зернышко (фильм второй)", "پروکسی مابعد", "青之花", "Heidi, la fille des Alpes",
// 		"银盘万花筒", "Temi d`amore tra i banchi di scuola", "Съюзът на Среброкрилите", "Аякаши",
// 		"Дух в оболонці: комплекс окремості", "贫乏姊妹物语", "La rose de Versailles",
// 		"แฮปปี้ เลสซั่น", "Juodasis Dievas", "Ерата Сенгоку: Последното парти",
// 		"Белина: Чезнеща в тъмнината", "Пламенный лабиринт", "Капризный Робот", "Kovboy Bebop: Film",
// 		"Bavel`in Kitabı", "东京魔人学院剑风帖 龙龙", "سكول رمبل الفصل الثاني", "青之驱魔师", "سايكانو",
// 		"神的记事本", "死神的歌谣", "Angel e a Flor de Sete Cores", "ماگی: هزارتوی جادو", "Spirală",
// 		"Chié la petite peste",
// 	}

// 	B.ResetTimer()
// 	for i := 0; i < B.N; i++ {
// 		db.FuzzySearch(vec[i%len(vec)])
// 	}
// }

// match of initial words
func BenchmarkFuzzySearch_initialWords(B *testing.B) {
	// cat anime-titles.dat | cut -d'|' -f4 | grep -E '[^ ]+ [^ ]+ [^ ]+' | \
	//     sort -R | cut -d' ' -f1,2 | sed 's/\(.*\)/"\1",/' | head -n 30
	vec := []string{
		"To Love", "Utawarerumono -", "Eden of", "D.C.if ～ダ・カーポ", "Вечност над",
		"Rupan Sansei:", "Los Caballeros", "Neko Hiki", "LoGH: A", "Arcadia of",
		"Pokémon 4Ever:", "Lenda Lunar", "Transformers: Master", "Tάρο, ο", "El Puño",
		"El taxi", "Lupin the", "Ah! My", "Le journal", "Odin: Koushi", "Amazing-man: The",
		"Legend of", "Youka no", "Я люблю", "Abe George", "Sisters of", "Ouran High",
		"Batman: Gotham", "Dantalian no", "Koi to", "Night Shift",
	}

	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		db.FuzzySearch(vec[i%len(vec)])
	}
}

// match of final words
func BenchmarkFuzzySearch_finalWords(B *testing.B) {
	// cat anime-titles.dat | cut -d'|' -f4 | grep -E '^[^ ]+ [^ ]+ [^ ]+ [^ ]+$' | \
	//     sort -R | cut -d' ' -f3,4 | sed 's/\(.*\)/"\1",/' | head -n 30
	vec := []string{
		"do Zodíaco", "Formula 91", "Shuto Houkai", "Deadly Sins", "gui lai",
		"muistoja tulevaisuudesta", "Mission 1-3", "スペシャルエディションII それぞれの剣", "Một Giây",
		"Meia-Lua Acima", "Mighty: Decode", "To Screw", "do Tênis", "(Duke Fleed)", "Olympic Taikai",
		"Драма ангелов", "Shihosha Judge", "демонов Йоко", "Shoujo Club", "Family (2)", "do Tesouro",
		"Witte Leeuw", "von Mandraguar", "Jin Xia", "Tabi Movie", "Symphonia 2", "no Tenkousei",
		"Movie (2011)", "Guardian Signs", "Você 2",
	}

	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		db.FuzzySearch(vec[i%len(vec)])
	}
}

// XXX: This is somehow the most time-consuming case, despite terminating several
// regular expressions earlier than the next two benchmarks.
//
// All regular expressions checked here (besides the .*-peppered one for initial condidate search)
// have no metacharacters at all besides the trivial \A and \z; while the ones for the following
// cases include more complicated grouped expressions...
func BenchmarkFuzzySearch_infixWords(B *testing.B) {
	// cat anime-titles.dat | cut -d'|' -f4 | grep -E '^[^ ]+ [^ ]+ [^ ]+ [^ ]+$' | \
	//     sort -R | cut -d' ' -f2,3 | sed 's/\(.*\)/"\1",/' | head -n 30
	vec := []string{
		"Yes! プリキュア5GoGo!", "Grime X-Rated", "Diễn Ngàn", "Super-Refined Ninja",
		"o Haita", "Conan: 14.", "the Seagulls", "009 Kaijuu", "Monogatari Daini-hen:",
		"no Haha", "по Ловец", "Centimeters per", "wang gui", "the Wandering", "Saru Kani",
		"Dark Red", "Pair: Project", "Охотник на", "trois petits", "of Teacher", "wa Suitai",
		"Lolita Fantasy", "εκατοστά το", "Eri-sama Katsudou", "希望の学園と絶望の高校生 The",
		"Comet SPT", "HUNTER スペシャル", "no Makemono", "Kızı: İkinci", "Pirate Captain",
	}

	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		db.FuzzySearch(vec[i%len(vec)])
	}
}

func BenchmarkFuzzySearch_alternatingWords(B *testing.B) {
	// cat anime-titles.dat | cut -d'|' -f4 | grep -E '^[^ ]+ [^ ]+ [^ ]+ [^ ]+ [^ ]+$' | \
	//     sort -R | cut -d' ' -f2,4 | sed 's/\(.*\)/"\1",/' | head -n 30
	vec := []string{
		"of Millennium", "Kreuz: und", "для Літнє", "Saikyou Deshi", "Hearts: no", "Roh Wolf",
		"III: Columbus", "Shin-chan Film", "Ball Superandroid", "恋のステージ=HEART FIRE!",
		"Disease Moon", "Corps Mecha", "BLOOD-C Last", "- trésor", "Lover a", "dievčati, preskočilo",
		"Star: Szomorú", "Ai Marchen", "Kishin &", "Seiya: Goddess", "Orange Shiroi", "Punch Sekai:",
		"No.1: no", "ο του", "プリキュアオールスターズ Stage", "Ankoku Hakai", "8-ма по", "II Ultimate",
		"Tenma Kuro", "Grade Kakusei",
	}

	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		db.FuzzySearch(vec[i%len(vec)])
	}
}

func BenchmarkFuzzySearch_worstCase(B *testing.B) {
	// cat anime-titles.dat | cut -d'|' -f4 | \
	//     perl -MEncode \
	//         -pe'chomp; $_ = encode_utf8(substr(decode_utf8($_), 1, -1) . "\n")' | \
	//     sort -R | sed 's/\(.*\)/"\1",/' | head -n 30
	// further perturbed by hand
	vec := []string{
		"ig ray S in han: Den tsu o Yob Amig",
		"ar Ben th Sea: 20.00 Mil for Lov",
		"eminin Famil",
		"界の断",
		"凹内かっぱまつ",
		"ゅーぶら!",
		"unog",
		"aji no ppo: pion Roa",
		"etect boy ma",
		"aruto Movi",
		"光のピア ユメミと銀 バラ騎士",
		"ki ru Sh j",
		"aint : Ο Χαμέ μβάς - Μυθολογία Άδ",
		"as Camarer s Mágica",
		"oll Be Foreve",
		"RAG BALL SODE of BAR",
		"ero eroppi no ken: Pink no",
		"acre east chin Cyg",
		"ister Princes",
		"PRINTS IN SAND",
		"е й хазяї",
		"quent in Dra",
		"inoc chio Bouke",
		"rm Libra : Banto",
		"2 sk sbrutna pojkar äventyrens",
		"タス",
		"last kinė Mažyl",
		"女チャングム 夢 第二",
		"錬金術師 嘆きの丘 の聖なる",
		"hou Rouge Lip"}

	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		db.FuzzySearch(vec[i%len(vec)])
	}
}
