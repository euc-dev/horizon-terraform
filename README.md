Plugin for Terraform Provider for Horizon View

Omnissa® has developed a custom Terraform provider for automating Horizon View Servers Installation and Upgrade. Using Terraform with Horizon View provider, you can deploy Horizon servers or upgrade the existing instance. The provider is developed and maintained by Omnissa® . Please note that this provider is still in tech preview.


Plugin for Terraform Provider for Horizon View Documentation

  Navigating the repository

  •internal folder - Contains the Horizon View provider implementation for Terraform

  •sample-workflows - Contains the examples for users to use Horizon View custom provider for install and upgrade of Horizon infrastructure.



In the current release the provider supports the below list of actions:

  . Creating a new role for Horizon Server Lifecycle Management
  
  . Assign the new role to specified horizon administrator
  
  . Register Horizon Server Package
  
  . Install Horizon Connection Server, Replica Server and Enrollment Server
  
  . Upgrade Horizon Connection Server, Replica Server and Enrollment Server
   


License

This project is Licensed under the Mozilla Public License Version 2.0 ; you may not use this file except in compliance with the License. 

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
