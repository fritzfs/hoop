(ns webapp.connections.views.connection-list
  (:require ["lucide-react" :refer [Wifi EllipsisVertical InfoIcon]]
            ["@radix-ui/themes" :refer [IconButton Box Button DropdownMenu Tooltip
                                        Flex Text Callout]]
            [clojure.string :as cs]
            [re-frame.core :as rf]
            [reagent.core :as r]
            [webapp.components.button :as button]
            [webapp.components.loaders :as loaders]
            [webapp.components.searchbox :as searchbox]
            [webapp.connections.constants :as connection-constants]
            [webapp.connections.views.connection-settings-modal :as connection-settings-modal]
            [webapp.config :as config]))

(defn empty-list-view []
  [:div {:class "pt-x-large"}
   [:figure
    {:class "w-1/6 mx-auto p-regular"}
    [:img {:src (str config/webapp-url "/images/illustrations/pc.svg")
           :class "w-full"}]]
   [:div {:class "px-large text-center"}
    [:div {:class "text-gray-700 text-sm font-bold"}
     "Beep boop, no sessions to look"]
    [:div {:class "text-gray-500 text-xs mb-large"}
     "There's nothing with this criteria"]]])


(defn- loading-list-view []
  [:div {:class "flex items-center justify-center rounded-lg border bg-white h-full"}
   [:div {:class "flex items-center justify-center h-full"}
    [loaders/simple-loader]]])

(defn aws-connect-sync-callout []
  (let [aws-jobs-running? @(rf/subscribe [:jobs/aws-connect-running?])]
    (when aws-jobs-running?
      [:> Callout.Root {:class "my-4"}
       [:> Callout.Icon
        [:> InfoIcon {:size 16}]]
       [:> Callout.Text
        [:> Text {:weight "bold" :as "span"} "AWS Connect Sync in Progress"]
        [:> Text {:as "span"} " There is an automated process for your connections happening in your hoop.dev environment. Check it later in order to verify."]]])))

(defn panel [_]
  (let [connections (rf/subscribe [:connections])
        user (rf/subscribe [:users->current-user])
        search-focused (r/atom false)
        searched-connections (r/atom nil)
        searched-criteria-connections (r/atom "")
        connections-search-status (r/atom nil)]
    (rf/dispatch [:connections->get-connections])
    (rf/dispatch [:users->get-user])
    (rf/dispatch [:guardrails->get-all])
    (rf/dispatch [:jobs/start-aws-connect-polling])
    (fn []
      (let [connections-search-results (if (empty? @searched-connections)
                                         (:results @connections)
                                         @searched-connections)]
        [:div {:class "flex flex-col bg-white rounded-lg h-full p-6 overflow-y-auto"}
         (when (-> @user :data :admin?)
           [:div {:class "absolute top-10 right-4 sm:right-6 lg:top-16 lg:right-20"}
            [:> Button {:on-click (fn [] (rf/dispatch [:navigate :create-connection]))}
             "Add Connection"]])
         [:> Flex {:as "header"
                   :direction "column"
                   :gap "3"
                   :class "mb-4"}
          [searchbox/main
           {:options (:results @connections)
            :display-key :name
            :searchable-keys [:name :type :subtype :connection_tags :status]
            :on-change-results-cb #(reset! searched-connections %)
            :hide-results-list true
            :placeholder "Search by connection name, type, status, tags or anything"
            :on-focus #(reset! search-focused true)
            :on-blur #(reset! search-focused false)
            :name "connection-search"
            :on-change #(reset! searched-criteria-connections %)
            :loading? (= @connections-search-status :loading)
            :size :small
            :icon-position "left"}]

          [aws-connect-sync-callout]]

         (if (and (= :loading (:status @connections)) (empty? (:results @connections)))
           [loading-list-view]

           [:div {:class "rounded-lg border bg-white h-full overflow-y-auto"}
            [:div {:class "relative h-full overflow-y-auto"}
                ;;  (when (and (= status :loading) (empty? (:data sessions)))
                ;;    [loading-list-view])
             (when (and (empty? (:results  @connections)) (not= (:status @connections) :loading))
               [empty-list-view])

             (if (and (empty? @searched-connections)
                      (> (count @searched-criteria-connections) 0))
               [:div {:class "px-regular py-large text-xs text-gray-700 italic"}
                "No connections with this criteria"]

               (doall
                (for [connection connections-search-results]
                  ^{:key (:id connection)}
                  [:div {:class (str "border-b last:border-0 hover:bg-gray-50 text-gray-700 "
                                     " p-regular text-xs flex gap-8 justify-between items-center")}
                   [:div {:class "flex truncate items-center gap-regular"}
                    [:div
                     [:figure {:class "w-5"}
                      [:img {:src  (connection-constants/get-connection-icon connection)
                             :class "w-9"}]]]
                    [:span {:class "block truncate"}
                     (:name connection)]]
                   [:div {:id "connection-info"
                          :class "flex gap-6 items-center"}

                    [:div {:class "flex items-center gap-1 text-xs text-gray-700"}
                     [:div {:class (str "rounded-full h-[6px] w-[6px] "
                                        (if (= (:status connection) "online")
                                          "bg-green-500"
                                          "bg-red-500"))}]
                     (cs/capitalize (:status connection))]

                    (when (or
                           (= "database" (:type connection))
                           (and (= "application" (:type connection))
                                (= "tcp" (:subtype connection))))
                      [:div {:class "relative cursor-pointer group"
                             :on-click #(rf/dispatch [:modal->open {:content [connection-settings-modal/main (:name connection)]
                                                                    :maxWidth "446px"}])}
                       [:> Tooltip {:content "Hoop Access"}
                        [:> IconButton {:size 1 :variant "ghost" :color "gray"}
                         [:> Wifi {:size 16}]]]])

                    [:> DropdownMenu.Root {:dir "rtl"}
                     [:> DropdownMenu.Trigger
                      [:> IconButton {:size 1 :variant "ghost" :color "gray"}
                       [:> EllipsisVertical {:size 16}]]]
                     [:> DropdownMenu.Content
                      (when (and (-> @user :data :admin?)
                                 (not (= (:managed_by connection) "hoopagent")))
                        [:> DropdownMenu.Item {:on-click
                                               (fn []
                                                 (rf/dispatch [:plugins->get-my-plugins])
                                                 (rf/dispatch [:navigate :edit-connection {} :connection-name (:name connection)]))}
                         "Configure"])
                      [:> DropdownMenu.Item {:color "red"
                                             :on-click (fn []
                                                         (rf/dispatch [:dialog->open
                                                                       {:title "Delete connection?"
                                                                        :type :danger
                                                                        :text-action-button "Confirm and delete"
                                                                        :action-button? true
                                                                        :text [:> Box {:class "space-y-radix-4"}
                                                                               [:> Text {:as "p"}
                                                                                "This action will instantly remove your access to "
                                                                                (:name connection)
                                                                                " and can not be undone."]
                                                                               [:> Text {:as "p"}
                                                                                "Are you sure you want to delete this connection?"]]
                                                                        :on-success (fn []
                                                                                      (rf/dispatch [:connections->delete-connection (:name connection)])
                                                                                      (rf/dispatch [:modal->close]))}]))}
                       "Delete"]]]]])))]])]))))
