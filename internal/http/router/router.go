package router

import (
	auth2 "monitoring_backend/internal/auth"
	"monitoring_backend/internal/http/handlers"
	"monitoring_backend/internal/http/handlers/auth"
	"monitoring_backend/internal/http/handlers/department"
	"monitoring_backend/internal/http/handlers/group"
	lecture2 "monitoring_backend/internal/http/handlers/lecture"
	"monitoring_backend/internal/http/handlers/practice"
	"monitoring_backend/internal/http/handlers/service/dataset"
	"monitoring_backend/internal/http/handlers/student_group"
	"monitoring_backend/internal/http/handlers/subject"
	"monitoring_backend/internal/http/handlers/user"
	"monitoring_backend/internal/http/middleware"
	"monitoring_backend/internal/http/response"
	"monitoring_backend/internal/lecture"
	"monitoring_backend/internal/ws"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Dependencies struct {
	Health *handlers.Handler

	AuthHandler *auth.AuthHandler

	Department   *department.DepartmentHandler
	Group        *group.GroupHandler
	StudentGroup *student_group.StudentGroupHandler
	Subject      *subject.SubjectHandler
	Lecture      *lecture2.LectureHandler
	Practice     *practice.PracticeHandler
	User         *user.UserHandler

	DataSet *dataset.DatasetHandler

	WsHub          *ws.Hub
	LectureManager *lecture.Manager
	JWTManager     *auth2.JWTManager
}

func New(d Dependencies) *mux.Router {
	r := mux.NewRouter()

	jwtMW := middleware.JWT(d.JWTManager)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, http.StatusNotFound, "not_found")
	})

	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/health", d.Health.Health).Methods(http.MethodGet)
	api.HandleFunc("/ws", ws.Handler(d.WsHub))

	// auth
	authGroup := r.PathPrefix("/auth").Subrouter()
	authGroup.HandleFunc("/login", d.AuthHandler.Login).Methods(http.MethodPost)

	// lectures
	lectureGroup := api.PathPrefix("/lecture").Subrouter()
	lectureGroup.HandleFunc("/start", d.LectureManager.StartLecture).Methods(http.MethodPost)
	lectureGroup.HandleFunc("/stop", d.LectureManager.StopLecture).Methods(http.MethodPost)

	api.HandleFunc("/departments", d.Department.List).Methods("GET")
	api.HandleFunc("/departments/{id:[0-9]+}", d.Department.GetByID).Methods("GET")
	api.HandleFunc("/departments/code/{code}", d.Department.GetByCode).Methods("GET")

	api.HandleFunc("/departments/{department_id:[0-9]+}/groups", d.Group.ListByDepartment).Methods("GET")
	api.HandleFunc("/groups/{code}", d.Group.GetByCode).Methods("GET")

	api.HandleFunc("/students/{isu}/group", d.StudentGroup.SetUserGroup).Methods("PUT")
	api.HandleFunc("/students/{isu}/group", d.StudentGroup.GetUserGroup).Methods("GET")
	api.HandleFunc("/students/{isu}/group", d.StudentGroup.RemoveUserGroup).Methods("DELETE")
	api.HandleFunc("/groups/{code}/students", d.StudentGroup.ListUsersByGroup).Methods("GET")

	api.HandleFunc("/subjects", d.Subject.Create).Methods("POST")
	api.HandleFunc("/subjects", d.Subject.List).Methods("GET")
	api.HandleFunc("/subjects/{id:[0-9]+}", d.Subject.GetByID).Methods("GET")
	api.HandleFunc("/subjects/by-name/{name}", d.Subject.GetByName).Methods("GET")

	api.HandleFunc("/lectures", d.Lecture.Create).Methods("POST")
	api.HandleFunc("/lectures/{id:[0-9]+}", d.Lecture.GetByID).Methods("GET")
	api.HandleFunc("/teachers/{isu}/lectures", d.Lecture.ListByTeacher).Methods("GET")
	api.HandleFunc("/subjects/{id:[0-9]+}/lectures", d.Lecture.ListBySubject).Methods("GET")
	api.HandleFunc("/groups/{code}/lectures", d.Lecture.ListByGroup).Methods("GET")

	api.HandleFunc("/practices", d.Practice.Create).Methods("POST")
	api.HandleFunc("/practices/{id:[0-9]+}", d.Practice.GetByID).Methods("GET")
	api.HandleFunc("/teachers/{isu}/practices", d.Practice.ListByTeacher).Methods("GET")
	api.HandleFunc("/subjects/{id:[0-9]+}/practices", d.Practice.ListBySubject).Methods("GET")
	api.HandleFunc("/groups/{code}/practices", d.Practice.ListByGroup).Methods("GET")

	api.HandleFunc("/lecture/start", d.LectureManager.StartLecture).Methods(http.MethodPost)
	api.HandleFunc("/lecture/stop", d.LectureManager.StopLecture).Methods(http.MethodPost)

	// cores
	userGroup := api.PathPrefix("/user").Subrouter()
	adminGroup := api.PathPrefix("/admin").Subrouter()
	adminGroup.Use(jwtMW)

	adminGroup.HandleFunc("/create", d.User.AddUser).Methods(http.MethodPost)
	userGroup.HandleFunc("/upload/faces/{isu}", d.User.UploadFaces).Methods(http.MethodPost)
	userGroup.HandleFunc("/roles", d.User.GetRoles).Methods(http.MethodGet)
	adminGroup.HandleFunc("/roles", d.User.GetRoles).Methods(http.MethodPost)

	// services
	serviceGroup := api.PathPrefix("/service").Subrouter()
	serviceGroup.HandleFunc("/dataset", d.DataSet.Get).Methods(http.MethodGet)

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return r
}
