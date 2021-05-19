# data codefresh_idps
```
data "codefresh_idps" "idp_azure" {
  display_name = "codefresh-onprem-tst-2"
  # client_name = "2222"
  # _id = "5df234543"
  client_type = "azure"
}

data "codefresh_idps" "local" {
  display_name = "local"
}

resource "codefresh_account" "acc" {
  name = "acc1"

  features = var.accountFeatures

  limits {
    collaborators = 25
    data_retention_weeks = 5
  }

  build {
    parallel = 25
    nodes = 7
  }

}

resource "codefresh_user" "user1" {
  email = "user1@example.com"
  user_name = "user1"

  activate = true

  roles = [
    "Admin",
    "User"
  ]

  login {
      idp_id = data.codefresh_idps.idp_azure.id
      sso = true
  }
  
  login  {
      idp_id = data.codefresh_idps.local.id
      //sso = false
  }

  personal {
    first_name = "John"
    last_name = "Smith"
  }

  accounts = [
    codefresh_account.acc.id
  ]
}

resource "codefresh_idp_accounts" "acc_idp" {
  idp_id = data.codefresh_idps.idp_azure.id
  account_ids = [codefresh_account.acc.id]
}
```