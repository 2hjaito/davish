case $PM in
  apk) pkg certbot certbot-nginx ;;
  *) pkg certbot python3-certbot-nginx ;;
esac
