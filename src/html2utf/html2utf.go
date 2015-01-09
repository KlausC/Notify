package main

import (
	"bufio"
	"bytes"
	"flag"
	_ "fmt"
	"io"
	"os"
)

var trans = [][]string{
	{"quot", "\""},
	{"amp", "&"},
	{"apos", "'"},
	{"lt", "<"},
	{"gt", ">"},
	{"nbsp", "\u00A0"},
	{"iexcl", "¡"},
	{"cent", "¢"},
	{"pound", "£"},
	{"curren", "¤"},
	{"yen", "¥"},
	{"brvbar", "¦"},
	{"sect", "§"},
	{"uml", "¨"},
	{"copy", "©"},
	{"ordf", "ª"},
	{"laquo", "«"},
	{"not", "¬"},
	{"shy", "\u00AD"},
	{"reg", "®"},
	{"macr", "¯"},
	{"deg", "°"},
	{"plusmn", "±"},
	{"sup2", "²"},
	{"sup3", "³"},
	{"acute", "´"},
	{"micro", "µ"},
	{"para", "¶"},
	{"middot", "·"},
	{"cedil", "¸"},
	{"sup1", "¹"},
	{"ordm", "º"},
	{"raquo", "»"},
	{"frac14", "¼"},
	{"frac12", "½"},
	{"frac34", "¾"},
	{"iquest", "¿"},
	{"Agrave", "À"},
	{"Aacute", "Á"},
	{"Acirc", "Â"},
	{"Atilde", "Ã"},
	{"Auml", "Ä"},
	{"Aring", "Å"},
	{"AElig", "Æ"},
	{"Ccedil", "Ç"},
	{"Egrave", "È"},
	{"Eacute", "É"},
	{"Ecirc", "Ê"},
	{"Euml", "Ë"},
	{"Igrave", "Ì"},
	{"Iacute", "Í"},
	{"Icirc", "Î"},
	{"Iuml", "Ï"},
	{"ETH", "Ð"},
	{"Ntilde", "Ñ"},
	{"Ograve", "Ò"},
	{"Oacute", "Ó"},
	{"Ocirc", "Ô"},
	{"Otilde", "Õ"},
	{"Ouml", "Ö"},
	{"times", "×"},
	{"Oslash", "Ø"},
	{"Ugrave", "Ù"},
	{"Uacute", "Ú"},
	{"Ucirc", "Û"},
	{"Uuml", "Ü"},
	{"Yacute", "Ý"},
	{"THORN", "Þ"},
	{"szlig", "ß"},
	{"agrave", "à"},
	{"aacute", "á"},
	{"acirc", "â"},
	{"atilde", "ã"},
	{"auml", "ä"},
	{"aring", "å"},
	{"aelig", "æ"},
	{"ccedil", "ç"},
	{"egrave", "è"},
	{"eacute", "é"},
	{"ecirc", "ê"},
	{"euml", "ë"},
	{"igrave", "ì"},
	{"iacute", "í"},
	{"icirc", "î"},
	{"iuml", "ï"},
	{"eth", "ð"},
	{"ntilde", "ñ"},
	{"ograve", "ò"},
	{"oacute", "ó"},
	{"ocirc", "ô"},
	{"otilde", "õ"},
	{"ouml", "ö"},
	{"divide", "÷"},
	{"oslash", "ø"},
	{"ugrave", "ù"},
	{"uacute", "ú"},
	{"ucirc", "û"},
	{"uuml", "ü"},
	{"yacute", "ý"},
	{"thorn", "þ"},
	{"yuml", "ÿ"},
	{"OElig", "Œ"},
	{"oelig", "œ"},
	{"Scaron", "Š"},
	{"scaron", "š"},
	{"Yuml", "Ÿ"},
	{"fnof", "ƒ"},
	{"circ", "ˆ"},
	{"tilde", "˜"},
	{"Alpha", "Α"},
	{"Beta", "Β"},
	{"Gamma", "Γ"},
	{"Delta", "Δ"},
	{"Epsilon", "Ε"},
	{"Zeta", "Ζ"},
	{"Eta", "Η"},
	{"Theta", "Θ"},
	{"Iota", "Ι"},
	{"Kappa", "Κ"},
	{"Lambda", "Λ"},
	{"Mu", "Μ"},
	{"Nu", "Ν"},
	{"Xi", "Ξ"},
	{"Omicron", "Ο"},
	{"Pi", "Π"},
	{"Rho", "Ρ"},
	{"Sigma", "Σ"},
	{"Tau", "Τ"},
	{"Upsilon", "Υ"},
	{"Phi", "Φ"},
	{"Chi", "Χ"},
	{"Psi", "Ψ"},
	{"Omega", "Ω"},
	{"alpha", "α"},
	{"beta", "β"},
	{"gamma", "γ"},
	{"delta", "δ"},
	{"epsilon", "ε"},
	{"zeta", "ζ"},
	{"eta", "η"},
	{"theta", "θ"},
	{"iota", "ι"},
	{"kappa", "κ"},
	{"lambda", "λ"},
	{"mu", "μ"},
	{"nu", "ν"},
	{"xi", "ξ"},
	{"omicron", "ο"},
	{"pi", "π"},
	{"rho", "ρ"},
	{"sigmaf", "ς"},
	{"sigma", "σ"},
	{"tau", "τ"},
	{"upsilon", "υ"},
	{"phi", "φ"},
	{"chi", "χ"},
	{"psi", "ψ"},
	{"omega", "ω"},
	{"thetasym", "ϑ"},
	{"upsih", "ϒ"},
	{"piv", "ϖ"},
	{"ensp", " "},
	{"emsp", " "},
	{"thinsp", " "},
	{"zwnj", "\u200C"},
	{"zwj", "\u200D"},
	{"lrm", "\u200E"},
	{"rlm", "\u200F"},
	{"ndash", "–"},
	{"mdash", "—"},
	{"lsquo", "‘"},
	{"rsquo", "’"},
	{"sbquo", "‚"},
	{"ldquo", "“"},
	{"rdquo", "”"},
	{"bdquo", "„"},
	{"dagger", "†"},
	{"Dagger", "‡"},
	{"bull", "•"},
	{"hellip", "…"},
	{"permil", "‰"},
	{"prime", "′"},
	{"Prime", "″"},
	{"lsaquo", "‹"},
	{"rsaquo", "›"},
	{"oline", "‾"},
	{"frasl", "⁄"},
	{"euro", "€"},
	{"image", "ℑ"},
	{"weierp", "℘"},
	{"real", "ℜ"},
	{"trade", "™"},
	{"alefsym", "ℵ"},
	{"larr", "←"},
	{"uarr", "↑"},
	{"rarr", "→"},
	{"darr", "↓"},
	{"harr", "↔"},
	{"crarr", "↵"},
	{"lArr", "⇐"},
	{"uArr", "⇑"},
	{"rArr", "⇒"},
	{"dArr", "⇓"},
	{"hArr", "⇔"},
	{"forall", "∀"},
	{"part", "∂"},
	{"exist", "∃"},
	{"empty", "∅"},
	{"nabla", "∇"},
	{"isin", "∈"},
	{"notin", "∉"},
	{"ni", "∋"},
	{"prod", "∏"},
	{"sum", "∑"},
	{"minus", "−"},
	{"lowast", "∗"},
	{"radic", "√"},
	{"prop", "∝"},
	{"infin", "∞"},
	{"ang", "∠"},
	{"and", "∧"},
	{"or", "∨"},
	{"cap", "∩"},
	{"cup", "∪"},
	{"int", "∫"},
	{"there4", "∴"},
	{"sim", "∼"},
	{"cong", "≅"},
	{"asymp", "≈"},
	{"ne", "≠"},
	{"equiv", "≡"},
	{"le", "≤"},
	{"ge", "≥"},
	{"sub", "⊂"},
	{"sup", "⊃"},
	{"nsub", "⊄"},
	{"sube", "⊆"},
	{"supe", "⊇"},
	{"oplus", "⊕"},
	{"otimes", "⊗"},
	{"perp", "⊥"},
	{"sdot", "⋅"},
	{"vellip", "⋮"},
	{"lceil", "⌈"},
	{"rceil", "⌉"},
	{"lfloor", "⌊"},
	{"rfloor", "⌋"},
	{"lang", "〈"},
	{"rang", "〉"},
	{"loz", "◊"},
	{"spades", "♠"},
	{"clubs", "♣"},
	{"hearts", "♥"},
	{"diams", "♦"}}

var emap *entityMap

type entityMap struct {
	emap map[string][]byte
}

func createEntityMap(html bool) *entityMap {
	emap := new(entityMap)
	emap.emap = make(map[string][]byte)
	for i, pair := range trans {
		if !(html && i <= 4) {
			key := pair[0]
			val := make([]byte, len(pair[1]))
			copy(val, pair[1])
			emap.emap[key] = val
		}
	}
	return emap
}
func entity(in []byte) (out []byte) {
	out = emap.emap[string(in)]
	return
}

var MAXENT int = 10

func splitEntities(data []byte, atEOF bool) (advance int, token []byte, err error) {
	ix := bytes.IndexByte(data, '&')
	advance = 0
	token = nil
	err = nil
	if ix < 0 {
		ix = len(data)
		if ix == 0 {
			return
		}
	}
	if ix > 0 {
		advance = ix
		token = data[0:ix]
		return
	}
	iz := bytes.IndexByte(data[1:], '&') + 1
	iy := bytes.IndexByte(data, ';')
	if iz > 0 && iy > 0 && iz <= iy {
		advance = iz
		token = data[0:iz]
		return
	}
	if iy > 0 {
		advance = iy + 1
		token = entity(data[1:iy])
		if len(token) == 0 {
			token = data[0 : iy+1]
		}
		return
	}
	if len(data) > MAXENT {
		advance = len(data)
		token = data
	}
	return
}

func readloop(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	scanner.Split(splitEntities)

	for scanner.Scan() {
		token := scanner.Bytes()
		//D fmt.Println(token)
		w.Write(token)
	}
}

var html = flag.Bool("html", false, "do not expand &(quot amp apos lt gt)")

func main() {
	flag.Parse()
	emap = createEntityMap(*html)
	readloop(os.Stdin, os.Stdout)
}
