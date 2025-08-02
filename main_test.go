package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
checkStatus:
Назначение: Вспомогательная функция для проверки статуса ответа
Описание: проверяет, что код ответа равен http.StatusOK
*/
func checkStatus(t *testing.T, response *httptest.ResponseRecorder) {
	// Проверяем, что статус ответа равен 200 OK
	require.Equal(t, http.StatusOK, response.Code)
}

/*
TestCafeNegative:
Назначение: тестирование негативных сценариев
Описание: проверяет обработку некорректных запросов, которые должны возвращать ошибки
*/
func TestCafeNegative(t *testing.T) {
	// Создаем HTTP-обработчик
	handler := http.HandlerFunc(mainHandle)

	// Слайс тестовых запросов с ожидаемыми результатами
	requests := []struct {
		request string // URL запроса
		status  int    // Ожидаемый статус ответа
		message string // Ожидаемое сообщение об ошибке
	}{
		{"/cafe", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=omsk", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=tula&count=na", http.StatusBadRequest, "incorrect count"},
	}

	// Итерируемся по всем тестовым запросам
	for _, v := range requests {
		// Создаем объект для записи ответа
		response := httptest.NewRecorder()

		// Создаем тестовый GET-запрос
		req := httptest.NewRequest("GET", v.request, nil)

		// Обрабатываем запрос
		handler.ServeHTTP(response, req)

		// Проверяем статус и сообщение об ошибке
		assert.Equal(t, v.status, response.Code)
		assert.Equal(t, v.message, strings.TrimSpace(response.Body.String()))
	}
}

/*
TestCafeWhenOk:
Назначение: тестирование позитивных сценариев
Описание: проверяет корректную работу обработчика при валидных параметрах запроса
*/
func TestCafeWhenOk(t *testing.T) {
	// Создаем HTTP-обработчик
	handler := http.HandlerFunc(mainHandle)

	// Слайс валидных тестовых запросов
	requests := []string{
		"/cafe?count=2&city=moscow",      // Запрос с указанием количества и города
		"/cafe?city=tula",                // Запрос только с городом
		"/cafe?city=moscow&search=ложка", // Запрос с поиском
	}

	// Итерируемся по всем тестовым запросам
	for _, v := range requests {
		// Создаем объект для записи ответа
		response := httptest.NewRecorder()

		// Создаем тестовый GET-запрос
		req := httptest.NewRequest("GET", v, nil)

		// Обрабатываем запрос
		handler.ServeHTTP(response, req)

		// Проверяем статус ответа
		checkStatus(t, response)
	}
}

/*
TestCafeCount:
Назначение: тестирование работы параметра count
Описание: проверяет корректность обработки различных значений параметра count
*/
func TestCafeCount(t *testing.T) {
	// Создаем HTTP-обработчик
	handler := http.HandlerFunc(mainHandle)

	// Слайс тестовых данных для проверки параметра count
	requests := []struct {
		count int // передаваемое значение count
		want  int // ожидаемое количество кафе в ответе
	}{
		{0, 0},     // пустой ответ
		{1, 1},     // один элемент
		{2, 2},     // два элемента
		{100, 100}, // Проверка максимального количества
	}

	// Итерируемся по всем тестовым данным
	for _, v := range requests {
		// И по всем городам в мапе cafeList из файла main.go
		for city := range cafeList {

			// Создаем объект для записи ответа
			response := httptest.NewRecorder()

			// Формируем URL запроса с нужными параметрами
			req := httptest.NewRequest("GET", fmt.Sprintf("/cafe?count=%d&city=%s", v.count, city), nil)

			// Обрабатываем запрос
			handler.ServeHTTP(response, req)

			// Проверяем статус ответа
			checkStatus(t, response)

			// Читаем тело ответа и проверяем отсутствие ошибок при чтении
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			// Для count=0 проверяем пустой ответ
			if v.count == 0 {
				require.Empty(t, body)
				continue
			}

			// Корректируем ожидаемое значение для count=100 если в городе меньше 100 кафе
			if v.count == 100 {
				v.want = min(v.count, len(cafeList[city]))
			}

			// Разбиваем ответ на слайс по разделителю ","
			res := strings.Split(string(body), ",")

			// Проверяем, что количество элементов соответствует ожидаемому
			assert.Equal(t, v.want, len(res))

		}
	}
}

/*
TestCafeSearch:
Назначение: тестирование работы параметра Search
Описание: Проверяется количество найденных кафе в зависимости от значений параметра Search
*/
func TestCafeSearch(t *testing.T) {
	// Создаем HTTP-обработчик
	handler := http.HandlerFunc(mainHandle)

	// Слайс тестовых данных для проверки параметра Search
	requests := []struct {
		search    string // значение параметра search для запроса
		wantCount int    // ожидаемое количество кафе в ответе
	}{
		{"фасоль", 0}, // ожидаем отсутствие кафе с таким словом
		{"кофе", 2},   // ожидаем 2 кафе с упоминанием слова "кофе"
		{"вилка", 1},  // ожидаем 1 кафе с упоминанием слова "вилка"
	}

	// Перебираем все города из списка кафе
	for city := range cafeList {
		// Для каждого города тестируем все поисковые запросы
		for _, v := range requests {

			// Создаем объект для записи ответа
			response := httptest.NewRecorder()

			// Формируем URL запроса с нужными параметрами
			req := httptest.NewRequest("GET", fmt.Sprintf("/cafe?city=%s&search=%s", city, v.search), nil)

			// Обрабатываем запрос
			handler.ServeHTTP(response, req)

			// Проверяем статус ответа
			checkStatus(t, response)

			// Читаем тело ответа и проверяем отсутствие ошибок при чтении
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			// Разбиваем ответ на слайс по разделителю ","
			res := strings.Split(strings.TrimSpace(string(body)), ",")

			// Проверяем каждый найденный результат
			for _, name := range res {
				trimmedName := strings.TrimSpace(name)
				if strings.Contains(strings.ToLower(trimmedName), strings.ToLower(v.search)) {
					assert.Equal(t, v.wantCount, len(res))
				}
			}
		}
	}
}
