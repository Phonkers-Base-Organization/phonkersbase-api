package repository

import "strings"

// likeEscaper escapes LIKE/ILIKE pattern metacharacters so user-supplied
// search terms match literally instead of acting as wildcards.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLikePattern(s string) string {
	return likeEscaper.Replace(s)
}
