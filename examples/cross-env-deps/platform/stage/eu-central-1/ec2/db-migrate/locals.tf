locals {
  path_arr = split("/", abspath(path.module))

  service     = element(local.path_arr, length(local.path_arr) - 5)
  environment = element(local.path_arr, length(local.path_arr) - 4)
  region      = element(local.path_arr, length(local.path_arr) - 3)
  scope       = element(local.path_arr, length(local.path_arr) - 2)
  module      = element(local.path_arr, length(local.path_arr) - 1)
}
