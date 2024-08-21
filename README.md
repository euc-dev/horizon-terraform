# Plugin for Terraform Provider for Horizon View

Omnissa® has developed a custom Terraform provider for automating Horizon View product deployment. Using Terraform with Horizon View provider, you can deploy Horizon servers or upgrade the existing instance. The provider is developed and maintained by Omnissa® . Please note that this provider is still in tech preview.

## Plugin for Terraform Provider for Horizon View Documentation

Navigating the repository

- internal folder - Contains the Horizon View provider implementation for Terraform
- sample-workflows - Contains the examples for users to use Horizon View custom provider for install and upgrade of Horizon infrastructure.

In the current release the provider supports the below list of actions:

- Creating a new role for Horizon Server Lifecycle Management
- Assign the new role to specified horizon administrator
- Register Horizon Server Package
- Install Horizon Connection Server, Replica Server and Enrollment Server
- Upgrade Horizon Connection Server, Replica Server and Enrollment Server

This repo is structured to feed into the developer.omnissa.com Developer Portal via the [](https://github.com/euc-dev/euc-dev.github.io) repo using MkDocs published by GitHub Pages. All documentation should be created in MarkDown format with the capabilities of MkDocs and the Material theme in mind.Only pages within the `/docs` folder should be modified within this repo.

This folder will be integrated into the [developer portal repo](https://github.com/euc-dev/euc-dev.github.io) when built using a GitHub Action.

## Downloads

By downloading, installing, or using the Software, you agree to be bound by the terms of Omnissa’s Software Development Kit License Agreement unless there is a different license provided in or specifically referenced by the downloaded file or package. If you disagree with any terms of the agreement, then do not use the Software.

## License

This project is licensed under the Creative Commons Attribution 4.0 International as described in [LICENSE](https://github.com/euc-dev/.github/blob/main/LICENSE); you may not use this file except in compliance with the License.

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
