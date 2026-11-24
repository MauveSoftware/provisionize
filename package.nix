{
  pkgs,
  lib,
  buildGo127Module,
}:

buildGo127Module {
  pname = "provisionize";
  version = "0.8.3";

  src = lib.cleanSource ./.;

  vendorHash = pkgs.lib.fileContents ./go.mod.sri;

  subPackages = [
    "cmd/provisionizer"
    "cmd/deprovisionizer"
  ];

  CGO_ENABLED = 0;

  meta = with lib; {
    description = "Zero touch provisioning for VMs with DNS integration";
    homepage = "https://github.com/MauveSoftware/provisionize";
    license = licenses.mit;
  };
}
