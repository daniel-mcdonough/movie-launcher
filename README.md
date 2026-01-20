# movie-launcher

A terminal UI for searching and launching videos from a local collection.

## Install

```
go install
```

Or build manually:
```
go build
```

## Usage

Set the `VIDEO_DIR` environment variable to your video directory:
```
export VIDEO_DIR=/path/to/videos
```

Optionally set a custom video player (defaults to mpv):
```
export VIDEO_PLAYER=vlc
```

Then search for videos:
```
movie-launcher matrix 1999
```

## Controls

- `j/k` or arrows - navigate
- `PgUp/PgDn` - page through results
- `g/G` - jump to top/bottom
- `/` - filter results
- `Enter` - play selected video
- `q` - quit
