pkg ufw
# The firewall is installed but left off: enabling it before allowing
# SSH would lock you out. Turn it on yourself with:
#   sudo ufw allow OpenSSH && sudo ufw enable
echo "ufw installed, not enabled. Allow SSH first: sudo ufw allow OpenSSH && sudo ufw enable"
