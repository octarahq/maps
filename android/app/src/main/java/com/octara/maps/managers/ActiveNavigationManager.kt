package com.octara.maps.managers

import android.content.Context
import com.google.gson.Gson
import com.octara.maps.types.ParsedTrip
import java.time.Instant
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.LocalTime
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZonedDateTime

object ActiveNavigationManager {
    private const val PREFS_NAME = "active_navigation_prefs"
    private const val KEY_TRIP_JSON = "active_trip_json"
    private const val KEY_SAVED_AT = "active_saved_at"

    private val gson = Gson()

    fun saveActiveNavigation(context: Context, trip: ParsedTrip) {
        val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit()
            .putString(KEY_TRIP_JSON, gson.toJson(trip))
            .putLong(KEY_SAVED_AT, System.currentTimeMillis())
            .apply()
    }

    fun clearActiveNavigation(context: Context) {
        val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit().clear().apply()
    }

    fun loadActiveNavigationIfStillValid(context: Context): ParsedTrip? {
        val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val json = prefs.getString(KEY_TRIP_JSON, null) ?: return null
        val savedAt = prefs.getLong(KEY_SAVED_AT, 0L)

        val trip = runCatching { gson.fromJson(json, ParsedTrip::class.java) }.getOrNull()
        if (trip == null || trip.legs.isEmpty()) {
            clearActiveNavigation(context)
            return null
        }

        val now = System.currentTimeMillis()
        val arrivalMillis = parseArrivalMillis(trip.arrivalTime)
        val deadline = (arrivalMillis ?: (savedAt + 4 * 60 * 60 * 1000L)) + 15 * 60 * 1000L
        if (now > deadline) {
            clearActiveNavigation(context)
            return null
        }

        return trip
    }

    private fun parseArrivalMillis(time: String?): Long? {
        if (time.isNullOrBlank()) return null
        val trimmed = time.trim()
        return runCatching { OffsetDateTime.parse(trimmed).toInstant().toEpochMilli() }.getOrNull()
            ?: runCatching { ZonedDateTime.parse(trimmed).toInstant().toEpochMilli() }.getOrNull()
            ?: runCatching { Instant.parse(trimmed).toEpochMilli() }.getOrNull()
            ?: runCatching {
                LocalDateTime.parse(trimmed).atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()
            }.getOrNull()
            ?: runCatching {
                LocalTime.parse(trimmed).atDate(LocalDate.now()).atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()
            }.getOrNull()
    }
}
