qrest is a quick RESTful JSON server

[![Build Status](https://travis-ci.org/landaire/qrest.svg?branch=travis-ci)](https://travis-ci.org/landaire/qrest)

# Usage

Create a JSON file containing the data you'd like to be part of your server. An example file might look like:

    {
        "posts": [ { "id": 1, "title": "Foo" } ]
    }

Start qrest with this file as an argument:

    qrest db.json

Or in a docker container:

    $ docker build -t qrest .
    $ docker run --rm -p 3000:3000 qrest "db.json" # assuming db.json is in this source directory

This will create the following routes for you to use:

    POST /posts (creates a new post record)
    GET /posts (returns all post records)
    GET /posts/:id (returns a specific record)
    PUT /posts/:id (creates or updates a record with the specified ID)
    PATCH /posts/:id (updates a record with the specified ID)
    DELETE /posts/:id (deletes the specified record)
    
# License

This project is released under the MIT license.

