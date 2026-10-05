package assets

import "embed"

//go:embed ocr_prompt.md ocr_solution_prompt.md prompts/chapters/*.md prompts/cs408/*/*.md prompts/general.md prompts/chapter_router.md all:web
var Files embed.FS
