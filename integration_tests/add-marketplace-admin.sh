cudos-noded keys add marketplace-admin-account --keyring-backend test |& tee "${CUDOS_HOME}/marketplace-admin-account.wallet"
chmod 600 "${CUDOS_HOME}/marketplace-admin-account.wallet"
MARKETPLACE_ADMIN_ACCOUNT_ADDRESS=$(echo $KEYRING_OS_PASS | cudos-noded keys show marketplace-admin-account -a --keyring-backend test)
cudos-noded add-genesis-account $MARKETPLACE_ADMIN_ACCOUNT_ADDRESS "10cudosAdmin"

cat "${CUDOS_HOME}/config/genesis.json" | jq --arg marketplaceAdminAddress "$MARKETPLACE_ADMIN_ACCOUNT_ADDRESS" '.app_state.marketplace.params.admins += [$marketplaceAdminAddress]' > "${CUDOS_HOME}/config/tmp_genesis.json" && mv "${CUDOS_HOME}/config/tmp_genesis.json" "${CUDOS_HOME}/config/genesis.json"
