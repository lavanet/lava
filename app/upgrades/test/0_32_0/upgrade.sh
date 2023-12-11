UPGRADE_INFO='{"expedited_min_deposit":"10000ulava","expedited_threshold":"0.67","expedited_voting_period":"1s"}'
upgrade-test/lavad tx gov submit-legacy-proposal software-upgrade v0.32.0 --upgrade-height 200 --upgrade-info=$UPGRADE_INFO --deposit 200ulava --from alice --no-validate --title="upgrade" --description="upgrade" --fees 1ulava --yes
upgrade-test/lavad tx gov  vote 1 yes --from alice --fees 4ulava --yes
