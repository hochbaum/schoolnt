# Schooln't
This is a small and ugly Discord bot sending the current weekly incidence
of Miltenberg into our class guild.

## Running
You have to build the image yourself because I am too lazy to add an action for it.
```
$ docker build -t schoolnt .
$ docker run --name bot schoolnt \
    --timer="0 18 * * *" \
    --token="mycooldiscordtoken" \
    --channel="theidofmycoolchannel"
```