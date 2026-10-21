
for f in /scripts/*.sh; do
  [ -f "$f" ] || continue
  case "$f" in
      *init.sh) continue ;;  # skip this file
    esac
  echo "Applying $f"
  sh "$f"
done

