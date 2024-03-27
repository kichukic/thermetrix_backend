FROM golang:1.21.5
FROM node
WORKDIR /usr/src/app
RUN go build -o myapp
RUN npm i pm2 -g
COPY . .
EXPOSE  4001
CMD ["pm2-runtime","start"]
