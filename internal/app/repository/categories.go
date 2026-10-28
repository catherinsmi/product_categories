package repository

// Categories — фиксированный список категорий, доступных пользователю
// при создании карточки. Порядок = порядок в селекте /add.
var Categories = []string{
	"Макияж",
	"Уход за лицом",
	"Уход за телом",
	"Парфюм",
	"Волосы",
	"Другое",
}

// IsValidCategory проверяет, что значение из формы входит в Categories.
func IsValidCategory(c string) bool {
	for _, x := range Categories {
		if x == c {
			return true
		}
	}
	return false
}
