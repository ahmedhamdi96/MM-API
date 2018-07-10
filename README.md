# MM-API
*MovieMood-API*

## Project Overview
*GUC, Winter 2017, CSEN704*

This is a simple GoLang API that provides movies information. It provides information about movies, actors/actresses, and recommendations based on a movie provided by the user. It contacts the [TMDb](https://www.themoviedb.org/?language=en) API to get the information.

> [Deployed API](https://moviemood-chatbot.herokuapp.com/)

## Routes

* GET /welcome
> Responsible for creating a new session for the connecting user by generating a new unique ID (UUID) that will be used to reference him/her later on. This endpoint is public.

* POST /chat
> Receives messages from the user and responds with a follow-up message or an error if something didn’t go as expected. This endpoint is private; it requires an Authorization header containing a user’s UUID.

## Testing

The API can be tested by using the chatbot UI located [here](http://ramin0-chatbot-ui.herokuapp.com/). The URL in the UI should be replaced with the API's URL, which is: https://moviemood-chatbot.herokuapp.com/.
 
## Project Partners
*in alphabetical order*

* [Ahmed Zaki](https://github.com/Zakyyy)
* [Ebraheem Mohamed](https://github.com/Ebraheem1)
