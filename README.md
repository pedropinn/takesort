# TakeSort

Uma maneira facil de organizar suas filmagens. Copie tudo para uma pasta e o TakeSort se encarrega de organizar os arquivos para voce.

Monitora a pasta `/media/temp` e move cada arquivo para `/media/{YYYY}/{YYYY-MM-DD}/` conforme o tipo:


| Extensao | Destino |
|----------|---------|
| `.mp4` | `{YYYY-MM-DD}/` |
| `.jpg` | `{YYYY-MM-DD}/photos/jpeg/` |
| `.dng` | `{YYYY-MM-DD}/photos/raw/` |
| `.wav` | `{YYYY-MM-DD}/audio/` |
| `.lrf`, `.lrv` | `{YYYY-MM-DD}/proxy/` |
| `.thm`, `.srt` | Deletado |

## Docker Compose
```yaml
services:
  takesort:
    image: pedropinn/takesort:latest
    user: "3000:3000"
    volumes:
      - /mnt/hdd/media/videos:/media
    environment:
      - TAKESORT_DEBOUNCE_INTERVAL=2s
      - TAKESORT_LOG_LEVEL=info
    restart: unless-stopped
```

## Configuracao

| Variavel | Default | Descricao |
|----------|---------|-----------|
| `TAKESORT_WATCH_DIR` | `/media/temp` | Pasta monitorada para novos arquivos |
| `TAKESORT_MEDIA_DIR` | `/media` | Pasta raiz para arquivos organizados |
| `TAKESORT_DEBOUNCE_INTERVAL` | `2s` | Tempo entre verificacoes de estabilidade do arquivo |
| `TAKESORT_LOG_LEVEL` | `info` | Nivel de log (debug, info, warn, error) |

Para melhor performance, monte watch e media no mesmo volume/filesystem para que `os.Rename` funcione sem copiar bytes.
