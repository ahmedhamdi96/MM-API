package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"os"
	"log"

	"github.com/gorilla/mux"
	cors "github.com/heppu/simple-cors"
)

//JSON OBJECT type
type JSON map[string]interface{}

// User Structure
type User struct {
	UUID            string
	Recommendations [2]string
}

// Movie Structure
type Movie struct {
	ID          float64
	Title       string
	Overview    string
	ReleaseDate string
	VoteAverage float64
}

//Actor Structure
type Actor struct {
	ID           float64
	Name         string
	Birthday     string
	Deathday     string
	Biography    string
	Gender       string
	PlaceOfBirth string
	Movies       []Movie
}

//MessageWrapper Structure
type MessageWrapper struct {
	Message string `json: "message, omitempty"`
}

var users []User

// writeJSON Writes the JSON equivilant for data into ResponseWriter w
func writeJSON(w http.ResponseWriter, data JSON) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x%x%x%x%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func handle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Available Routes:\n\n"+
		"  GET  /welcome -> handleWelcome\n"+
		"  POST /chat    -> handleChat\n"+
		"  GET  /        -> handle        (current)\n")
}

func handleWelcome(w http.ResponseWriter, r *http.Request) {

	uuid, err := newUUID()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "The server can not generate a UUID now, try again later!")
	}
	
	w.Header().Set("Content-Type", "application/json")
	jsonData := map[string]string{"message": "Welcome to MovieMood.\nHere are the comands you can use: {Movie [MOVIE_NAME], Actor/Actress [ACTOR_NAME/ACTRESS_NAME], Suggest}", "uuid": uuid}
	users = append(users, User{UUID: uuid})
	json.NewEncoder(w).Encode(jsonData)

}

func handleChat(w http.ResponseWriter, r *http.Request) {

	var messageWrapper MessageWrapper
	json.NewDecoder(r.Body).Decode(&messageWrapper)

	authorizationHeader := r.Header.Get("Authorization")
	validAuthorizationHeader := false

	for _, item := range users {
		if item.UUID == authorizationHeader {
			validAuthorizationHeader = true
		}
	}

	if validAuthorizationHeader == true {
		result := stringMatching(messageWrapper.Message)
		error := result["error"]
		serverError := result["server-error"]

		response := make(map[string]interface{})
		response["message"] = result["message"]

		if (error == nil && serverError == nil){
			writeJSON(w, response)
		} else {
			if(error != nil){
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			fmt.Fprintln(w, response["message"])
		}
		
	} else {
		//We need to handle this by error handling
		w.WriteHeader(http.StatusUnauthorized)
		// invalidMessage := make(map[string]interface{})
		// invalidMessage["message"] = "Invalid Authorization Header! Navigate to the /welcome route to get authorized."
		// writeJSON(w, invalidMessage)
		fmt.Fprintln(w, "Invalid Authorization Header! Navigate to the /welcome route to get authorized.")
	}
}

func stringMatching(message string) JSON {
	words := strings.Split(message, " ")
	command := words[0]
	name := strings.TrimPrefix(message, command+" ")

	if len(words) >= 2 && (name=="" || words[1]==""){
		invalidMessage := make(map[string]interface{})
		invalidMessage["error"] = "Invalid Command! Please use the following commands: {Movie [MOVIE_NAME], Actor/Actress [ACTOR_NAME/ACTRESS_NAME], Suggest}"
		invalidMessage["message"] = "Invalid Command! Please use the following commands: {Movie [MOVIE_NAME], Actor/Actress [ACTOR_NAME/ACTRESS_NAME], Suggest}"
		return invalidMessage
	}

	if strings.ToLower(command) == "movie" {
		if len(words)<2 {
			invalidMessage := make(map[string]interface{})
			invalidMessage["error"] = "The movie name should consist of at least one word!"
			invalidMessage["message"] = "The movie name should consist of at least one word!"
			return invalidMessage
		}

		objectReturned := make(map[string]interface{})
		movies, errJSON := getMovie(handleSpaces(name))
		if errJSON != nil {
			return errJSON
		}
		moviesString := parseMovies(movies, false)

		if len(movies) != 0 {
			objectReturned["message"] = moviesString

		} else {
			objectReturned["message"] = "No Results!"
		}

		return objectReturned
	}

	if strings.ToLower(command) == "actor" || strings.ToLower(command) == "actress" {
		if len(words) != 3 {
			invalidMessage := make(map[string]interface{})
			invalidMessage["error"] = "The actor/actress should consist of a First Name and Last Name!"
			invalidMessage["message"] = "The actor/actress should consist of a First Name and Last Name!"
			return invalidMessage
		}

		objectReturned := make(map[string]interface{})
		actors, errJSON := getActor(handleSpaces(name), name)
		if errJSON != nil {
			return errJSON
		}
		actorsString := parseActors(actors)

		if len(actors) != 0 {
			objectReturned["message"] = actorsString

		} else {
			objectReturned["message"] = "No Results!"
		}

		return objectReturned
	}

	if strings.ToLower(command) == "suggest" {
		objectReturned := make(map[string]interface{})
		objectReturned["message"] = "A movie suggestion will be provided based on a favourite movie of yours. Use the following format: Favourite [Movie_Name]."
		return objectReturned
	}

	if strings.ToLower(command) == "favourite" {
		if len(words)<2 {
			invalidMessage := make(map[string]interface{})
			invalidMessage["error"] = "The movie name should consist of at least one word!"
			invalidMessage["message"] = "The movie name should consist of at least one word!"
			return invalidMessage
		}

		objectReturned := make(map[string]interface{})
		movies, errJSON := getRecommendation(handleSpaces(name))
		if errJSON != nil {
			return errJSON
		}
		moviesString := parseMovies(movies, true)

		if len(movies) != 0 {
			objectReturned["message"] = moviesString

		} else {
			objectReturned["message"] = "No Results!"
		}
		return objectReturned
	}

	invalidMessage := make(map[string]interface{})
	invalidMessage["error"] = "Invalid Command! Please use the following commands: {Movie [MOVIE_NAME], Actor/Actress [ACTOR_NAME/ACTRESS_NAME], Suggest}"
	invalidMessage["message"] = "Invalid Command! Please use the following commands: {Movie [MOVIE_NAME], Actor/Actress [ACTOR_NAME/ACTRESS_NAME], Suggest}"
	return invalidMessage
}

func getMovie(movie string) ([]Movie, JSON) {

	response, err := http.Get("https://api.themoviedb.org/3/search/movie?api_key=185a996898bc5f90934413d4f55ae50c&language=en-US&query=" + movie)

	if err != nil {
		invalidMessage := make(map[string]interface{})
		invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
		invalidMessage["message"] = invalidMessage["server-error"]
		return nil, invalidMessage
	} else {
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		//Converting the slice of bytes into map[string]interface{} to hold any generic data types as
		//values for the key string
		var responseData map[string]interface{}
		err := json.Unmarshal(data, &responseData)
		if err != nil {
			invalidMessage := make(map[string]interface{})
			invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
			invalidMessage["message"] = invalidMessage["server-error"]
			return nil, invalidMessage
		}
		var moviesSearched []Movie
		movieInstance := Movie{}
		//Here we convert the list of the movies into array of interfaces
		//so that we can extract the data to be saved in our Movie struct
		filmsData := responseData["results"].([]interface{})
		results := len(filmsData)
		i := 0
		for results > 0 {
			movie := filmsData[i].(map[string]interface{})
			//Here we type float as the id is stored in the api as float64
			movieInstance.ID = movie["id"].(float64)
			movieInstance.Overview = movie["overview"].(string)
			movieInstance.Title = movie["title"].(string)
			movieInstance.ReleaseDate = movie["release_date"].(string)
			//Here we type float as the id is stored in the api as float64
			movieInstance.VoteAverage = movie["vote_average"].(float64)
			moviesSearched = append(moviesSearched, movieInstance)
			i = i + 1
			results = results - 1
		}
		return moviesSearched, nil
	}
}
func getPersonalInfo(actor *Actor) JSON{
	response, err := http.Get("https://api.themoviedb.org/3/person/" + strconv.Itoa(int((*actor).ID)) + "?api_key=185a996898bc5f90934413d4f55ae50c&language=en-US")
	if err != nil {
		invalidMessage := make(map[string]interface{})
		invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
		invalidMessage["message"] = invalidMessage["server-error"]
		return invalidMessage
	} else {
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		var responseData map[string]interface{}
		err := json.Unmarshal(data, &responseData)
		if err != nil {
			invalidMessage := make(map[string]interface{})
			invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
			invalidMessage["message"] = invalidMessage["server-error"]
			return invalidMessage
		}
		if(responseData["birthday"] != nil){
			(*actor).Birthday = responseData["birthday"].(string)
		}
		if responseData["deathday"] != nil {
			(*actor).Deathday = responseData["deathday"].(string)
		}
		if(responseData["place_of_birth"] != nil){
			(*actor).PlaceOfBirth = responseData["place_of_birth"].(string)		
		}
		if(responseData["biography"] != nil){
			(*actor).Biography = responseData["biography"].(string)	
		}
		if(responseData["gender"] != nil){
			if responseData["gender"].(float64) == 1 {
				(*actor).Gender = "Female"
			} else {
				(*actor).Gender = "Male"
			}
		}
		return nil	
	}
}

func getActor(actorName string, actorFullName string) ([]Actor, JSON){

	response, err := http.Get("https://api.themoviedb.org/3/search/person?api_key=185a996898bc5f90934413d4f55ae50c&language=en-US&query=" + actorName)

	if err != nil {
		invalidMessage := make(map[string]interface{})
		invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
		invalidMessage["message"] = invalidMessage["server-error"]
		return nil, invalidMessage
	} else {
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		//Converting the slice of bytes into map[string]interface{} to hold any generic data types as
		//values for the key string
		var responseData map[string]interface{}
		err := json.Unmarshal(data, &responseData)
		if err != nil {
			invalidMessage := make(map[string]interface{})
			invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
			invalidMessage["message"] = invalidMessage["server-error"]
			return nil, invalidMessage
		}
		var actorSearched []Actor
		actorInstance := Actor{}
		//Here we convert the list of the movies into array of interfaces
		//so that we can extract the data to be saved in our Movie struct
		actorsData := responseData["results"].([]interface{})
		results := len(actorsData)
		movie := Movie{}
		i := 0
		for results > 0 {
			actor := actorsData[i].(map[string]interface{})
			if(strings.ToLower(actorFullName) == strings.ToLower(actor["name"].(string))){
				actorInstance.ID = actor["id"].(float64)
				actorInstance.Name = actor["name"].(string)
				movies := actor["known_for"].([]interface{})
				j := 0
				moviesLength := len(movies)
				//This for loop to get the well-known movies for that actor/actress
				for moviesLength > 0 {
					movieInstance := movies[j].(map[string]interface{})
					movie.Title = movieInstance["title"].(string)
					movie.Overview = movieInstance["overview"].(string)
					movie.ReleaseDate = movieInstance["release_date"].(string)
					movie.ID = movieInstance["id"].(float64)
					movie.VoteAverage = movieInstance["vote_average"].(float64)
					actorInstance.Movies = append(actorInstance.Movies, movie)
					moviesLength = moviesLength - 1
					j = j + 1
				}
				//We get the personal info of the actor/actress by initiating a new request to the API
				errJSON := getPersonalInfo(&actorInstance)
				if errJSON != nil {
					return nil, errJSON
				}
				actorSearched = append(actorSearched, actorInstance)
				break
			}
			results = results - 1
			i = i + 1
		}
		return actorSearched, nil
	}
}

func getRecommendationHelper(id float64, movies *[]Movie) JSON {

	response, err := http.Get("https://api.themoviedb.org/3/movie/" + strconv.Itoa(int(id)) + "/recommendations?api_key=185a996898bc5f90934413d4f55ae50c&language=en-US")
	if err != nil {
		invalidMessage := make(map[string]interface{})
		invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
		invalidMessage["message"] = invalidMessage["server-error"]
		return invalidMessage
	} else {
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		var responseData map[string]interface{}
		err := json.Unmarshal(data, &responseData)
		if err != nil {
			invalidMessage := make(map[string]interface{})
			invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
			invalidMessage["message"] = invalidMessage["server-error"]
			return invalidMessage
		}
		filmsData := responseData["results"].([]interface{})
		movieInstance := Movie{}
		results := len(filmsData)
		i := 0
		if results > 10 {
			results = 10
		}
		for results > 0 {
			movie := filmsData[i].(map[string]interface{})
			//Here we type float as the id is stored in the api as float64
			if movie["id"] != nil {
				movieInstance.ID = movie["id"].(float64)
			}
			if movie["overview"] != nil {
				movieInstance.Overview = movie["overview"].(string)
			}
			if movie["title"] != nil {
				movieInstance.Title = movie["title"].(string)
			}
			if movie["release_date"] != nil {
				movieInstance.ReleaseDate = movie["release_date"].(string)
			}
			if movie["vote_average"] != nil {
				//Here we type float as the id is stored in the api as float64
				movieInstance.VoteAverage = movie["vote_average"].(float64)	
			}
			
			*movies = append((*movies), movieInstance)
			i = i + 1
			results = results - 1
		}
		return nil
	}
}

func getRecommendation(movie string) ([]Movie, JSON) {

	response, err := http.Get("https://api.themoviedb.org/3/search/movie?api_key=185a996898bc5f90934413d4f55ae50c&language=en-US&query=" + movie)

	if err != nil {
		invalidMessage := make(map[string]interface{})
		invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
		invalidMessage["message"] = invalidMessage["server-error"]
		return nil, invalidMessage
	} else {
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		//Converting the slice of bytes into map[string]interface{} to hold any generic data types as
		//values for the key string
		var responseData map[string]interface{}
		err := json.Unmarshal(data, &responseData)
		if err != nil {
			invalidMessage := make(map[string]interface{})
			invalidMessage["server-error"] = "The server can not process your request right now, try again later!"
			invalidMessage["message"] = invalidMessage["server-error"]
			return nil, invalidMessage
		}
		var suggestedMovies []Movie
		//Here we convert the list of the movies into array of interfaces
		//so that we can extract the data to be saved in our Movie struct
		filmsData := responseData["results"].([]interface{})
		results := len(filmsData)
		if results > 0 {
			movie := filmsData[0].(map[string]interface{})
			//Here we type float as the id is stored in the api as float64
			movieID := movie["id"].(float64)
			errJSON := getRecommendationHelper(movieID, &suggestedMovies)
			if errJSON != nil {
				return nil, errJSON
			}
			return suggestedMovies, nil
		}
	}

	return nil,nil
}

func handleSpaces(s string) string {
	var sOUT string
	for _, value := range s {
		if string(value) == " " {
			sOUT += "%20"
		} else {
			sOUT += string(value)
		}
	}
	return sOUT
}

func parseMovies(movies []Movie, suggest bool) string {
	var moviesString string

	if(suggest){
		moviesString = "Found "+ strconv.Itoa(len(movies)) +" matching suggestions.\n"
	}else{
		moviesString = "Found "+ strconv.Itoa(len(movies)) +" matching results.\n"
	}

	for index, movie := range movies {
		if index == len(movies)-1 {
			moviesString+="Movie #" + strconv.Itoa(index+1) + ": {Title: " + movie.Title + ", Overview: " + movie.Overview + ", Rating: " + strconv.FormatFloat(movie.VoteAverage, 'f', -1, 64) + ", Release Date: " + movie.ReleaseDate + "}"
		} else {
			moviesString+="Movie #" + strconv.Itoa(index+1) + ": {Title: " + movie.Title + ", Overview: " + movie.Overview + ", Rating: " + strconv.FormatFloat(movie.VoteAverage, 'f', -1, 64) + ", Release Date: " + movie.ReleaseDate + "}\n"
		}
	}
	return moviesString
}

func parseMoviesTitles(movies []Movie) string {
	moviesString := "{"
	for index, movie := range movies {
		if index == len(movies)-1 {
			moviesString += movie.Title + "}"
		} else {
			moviesString += movie.Title + ", "
		}	
	}
	return moviesString
}

func parseActors(actors []Actor) string {
	actorsString := "Found "+ strconv.Itoa(len(actors)) +" matching results.\n"
	deathday := ""
	biography := ""
	for index, actor := range actors {
		if actor.Deathday != "" {
			deathday = ", Deathday: " + actor.Deathday
		}
		if actor.Biography != "" {
			biography = ", Biography: " + actor.Biography
		}
		if index == len(actors)-1 {
			actorsString+="Actor #" + strconv.Itoa(index+1) + ": {Name: " + actor.Name + ", Birthday: " + actor.Birthday + deathday + biography + ", Gender: " + actor.Gender + ", Place of Birth: "+actor.PlaceOfBirth + ", Known For: " + parseMoviesTitles(actor.Movies) + "}"
		} else {
			actorsString+="Actor #" + strconv.Itoa(index+1) + ": {Name: " + actor.Name + ", Birthday: " + actor.Birthday + deathday + biography + ", Gender: " + actor.Gender + ", Place of Birth: "+actor.PlaceOfBirth + ", Known For: " + parseMoviesTitles(actor.Movies) + "}\n"
		}
	}
	return actorsString
}

func main() {

	router := mux.NewRouter()

	router.HandleFunc("/", handle).Methods("GET")
	router.HandleFunc("/welcome", handleWelcome).Methods("GET")
	router.HandleFunc("/chat", handleChat).Methods("POST")

	port := os.Getenv("PORT")
	// Default to 3000 if no PORT environment variable was defined
	if port == "" {
		port = "3000"
	}

	// Start the server
	fmt.Printf("Server is up and running on port %s...\n", port)
	log.Fatalln(http.ListenAndServe(":" + port, cors.CORS(router)))
}
