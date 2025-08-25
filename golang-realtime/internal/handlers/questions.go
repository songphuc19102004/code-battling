package handlers

import (
	"golang-realtime/pkg/common/response"
	"net/http"
)

func (hr *HandlerRepo) ListQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	questions, err := hr.queries.ListQuestions(r.Context())
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, questions, false, "")
}
