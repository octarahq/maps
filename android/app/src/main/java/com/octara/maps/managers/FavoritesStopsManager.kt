package com.octara.maps.managers

import android.content.Context
import android.content.SharedPreferences
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import com.octara.maps.types.Route
import com.octara.maps.types.Stop
import kotlinx.coroutines.flow.MutableStateFlow

data class FavoriteStop(
    val stopName: String,
    val stopId: String,
    val routeName: String,
    val routeId: String,
    val routeColor: String,
    val routepicto: String? = null
)

class FavoritesStopsManager(context: Context? = null) {
    init {
        context?.let { initialize(it) }
    }

    val favoriteStops: MutableStateFlow<List<FavoriteStop>>
        get() = Companion.favoriteStops

    fun toggleFavoriteStop(stop: Stop, route: Route) {
        Companion.toggleFavoriteStop(stop, route)
    }

    fun toggleFavoriteStop(
        stopId: String,
        routeId: String,
        stopName: String = "",
        routeName: String = "",
        routeColor: String = "",
        routepicto: String? = null
    ) {
        Companion.toggleFavoriteStop(stopId, routeId, stopName, routeName, routeColor, routepicto)
    }

    fun isFavorite(stopId: String, routeId: String): Boolean {
        return Companion.isFavorite(stopId, routeId)
    }

    fun isFavorite(stop: Stop, route: Route): Boolean {
        return Companion.isFavorite(stop, route)
    }

    fun getAllFavorites(): List<FavoriteStop> {
        return Companion.getAllFavorites()
    }

    fun getFavorites(): List<FavoriteStop> {
        return Companion.getFavorites()
    }

    companion object {
        private const val PREFS_NAME = "octara_favorites_prefs"
        private const val KEY_FAVORITE_STOPS = "favorite_stops"

        private var sharedPreferences: SharedPreferences? = null
        private val gson = Gson()

        val favoriteStops: MutableStateFlow<List<FavoriteStop>> = MutableStateFlow(mutableListOf())

        fun initialize(context: Context) {
            if (sharedPreferences == null) {
                sharedPreferences = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
                loadFromPrefs()
            }
        }

        private fun loadFromPrefs() {
            val prefs = sharedPreferences ?: return
            val json = prefs.getString(KEY_FAVORITE_STOPS, null) ?: return
            try {
                val type = object : TypeToken<List<FavoriteStop>>() {}.type
                val loaded: List<FavoriteStop>? = gson.fromJson(json, type)
                if (loaded != null) {
                    favoriteStops.value = loaded
                }
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }

        private fun saveToPrefs(favorites: List<FavoriteStop>) {
            val prefs = sharedPreferences ?: return
            try {
                val json = gson.toJson(favorites)
                prefs.edit().putString(KEY_FAVORITE_STOPS, json).apply()
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }

        fun toggleFavoriteStop(stop: Stop, route: Route) {
            val favoriteStop = FavoriteStop(stop.name, stop.id, route.name, route.id, route.color, route.picto)
            val currentFavorites = favoriteStops.value.toMutableList()

            val exists = currentFavorites.any {
                (it.stopId == stop.id && it.routeId == route.id) ||
                        (it.stopName == stop.name && it.routeName == route.name)
            }
            if (exists) {
                currentFavorites.removeAll {
                    (it.stopId == stop.id && it.routeId == route.id) ||
                            (it.stopName == stop.name && it.routeName == route.name)
                }
            } else {
                currentFavorites.add(favoriteStop)
            }
            favoriteStops.value = currentFavorites
            saveToPrefs(currentFavorites)
        }

        fun toggleFavoriteStop(
            stopId: String,
            routeId: String,
            stopName: String = "",
            routeName: String = "",
            routeColor: String = "",
            routepicto: String? = null
        ) {
            val currentFavorites = favoriteStops.value.toMutableList()
            val exists = currentFavorites.any {
                (it.stopId == stopId && it.routeId == routeId) ||
                        (it.stopId.removePrefix("IDFM:") == stopId.removePrefix("IDFM:") &&
                                it.routeId.removePrefix("IDFM:") == routeId.removePrefix("IDFM:"))
            }
            if (exists) {
                currentFavorites.removeAll {
                    (it.stopId == stopId && it.routeId == routeId) ||
                            (it.stopId.removePrefix("IDFM:") == stopId.removePrefix("IDFM:") &&
                                    it.routeId.removePrefix("IDFM:") == routeId.removePrefix("IDFM:"))
                }
            } else {
                currentFavorites.add(
                    FavoriteStop(
                        stopName = stopName,
                        stopId = stopId,
                        routeName = routeName,
                        routeId = routeId,
                        routeColor = routeColor,
                        routepicto = routepicto
                    )
                )
            }
            favoriteStops.value = currentFavorites
            saveToPrefs(currentFavorites)
        }

        fun isFavorite(stopId: String, routeId: String): Boolean {
            val cleanStopId = stopId.removePrefix("IDFM:")
            val cleanRouteId = routeId.removePrefix("IDFM:")
            return favoriteStops.value.any {
                (it.stopId == stopId && it.routeId == routeId) ||
                        (it.stopId.removePrefix("IDFM:") == cleanStopId &&
                                it.routeId.removePrefix("IDFM:") == cleanRouteId)
            }
        }

        fun isFavorite(stop: Stop, route: Route): Boolean {
            return isFavorite(stop.id, route.id) || favoriteStops.value.any {
                it.stopName == stop.name && it.routeName == route.name
            }
        }

        fun getAllFavorites(): List<FavoriteStop> {
            return favoriteStops.value
        }

        fun getFavorites(): List<FavoriteStop> {
            return getAllFavorites()
        }
    }
}