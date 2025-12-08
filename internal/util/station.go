package util

import "fmt"

func CleanStationName(namaLokasi, namaAlat string) string {
    if namaAlat == "" || namaAlat == namaLokasi {
        return namaLokasi
    }

    words := splitWords(namaAlat)

    if len(words) <= 1 {
        return namaLokasi
    }

    nameStartIdx := 0
    for i, word := range words {
        if looksLikeStationID(word) {
            nameStartIdx = i + 1
        } else {
            break
        }
    }

    if nameStartIdx < len(words) {
        name := joinWords(words[nameStartIdx:])
        return fmt.Sprintf("%s %s", namaLokasi, name)
    }

    return namaLokasi
}

func splitWords(s string) []string {
    var words []string
    var current []rune

    for _, r := range s {
        if r == ' ' || r == '-' || r == '_' || r == '.' {
            if len(current) > 0 {
                words = append(words, string(current))
                current = nil
            }
        } else {
            current = append(current, r)
        }
    }

    if len(current) > 0 {
        words = append(words, string(current))
    }

    return words
}

func joinWords(words []string) string {
    result := ""
    for i, w := range words {
        if i > 0 {
            result += " "
        }
        result += w
    }
    return result
}

func looksLikeStationID(s string) bool {
    if len(s) == 0 {
        return false
    }

    hasLetter := false
    hasDigit := false

    for _, r := range s {
        if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
            hasLetter = true
        }
        if r >= '0' && r <= '9' {
            hasDigit = true
        }
    }

    if hasLetter && hasDigit {
        return true
    }

    if len(s) <= 4 && isUppercase(s) {
        return true
    }

    return false
}

func isUppercase(s string) bool {
    for _, r := range s {
        if r >= 'a' && r <= 'z' {
            return false
        }
    }
    return true
}