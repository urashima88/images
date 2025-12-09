package tags_info

import (
	"images/internal/lib/api/response"
	"images/internal/lib/api/tag"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Request struct {
	TagIDs []string `json:"tag_ids"`
}

type Response struct {
	response.Response
	Tags []tag.Tag `json:"tags"`
}

type TagInfoDBGetter interface {
	GetTagsByIDs([]string) ([]tag.Tag, error)
}

type UUIDService interface {
	CleanAndValidateIDs(ids []string) []string
}

func New(log *slog.Logger, tagInfoDBGetter TagInfoDBGetter, uuidService UUIDService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.tags.info.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request body"))
			return
		}

		if len(req.TagIDs) == 0 {
			log.Error("no tag_ids provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("at least one tag_id is required"))
			return
		}

		cleanedTagIDs := uuidService.CleanAndValidateIDs(req.TagIDs)
		if len(cleanedTagIDs) == 0 {
			log.Error("all tag_ids are invalid")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("all provided tag_ids are invalid"))
			return
		}

		log.Info("fetching tags info", slog.Int("tag_ids_count", len(cleanedTagIDs)))

		tagsInfo, err := tagInfoDBGetter.GetTagsByIDs(cleanedTagIDs)
		if err != nil {
			log.Error("failed to get tags info", slog.String("error", err.Error()))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get tags info"))
			return
		}

		if len(tagsInfo) == 0 {
			log.Info("no tags found", slog.Int("requested_count", len(cleanedTagIDs)))
			render.JSON(w, r, Response{
				Response: response.OK(),
				Tags:     []tag.Tag{},
			})
			return
		}

		tagsResponse := make([]tag.Tag, len(tagsInfo))
		for i, info := range tagsInfo {
			tagsResponse[i] = tag.Tag{
				ID:        info.ID,
				Name:      info.Name,
				CreatedAt: info.CreatedAt,
			}
		}

		log.Info("tags retrieved successfully",
			slog.Int("requested_count", len(cleanedTagIDs)),
			slog.Int("found_count", len(tagsResponse)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			Tags:     tagsResponse,
		})
	}
}
