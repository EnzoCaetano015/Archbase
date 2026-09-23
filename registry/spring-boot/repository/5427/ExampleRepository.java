package com.example.api.repository;

import com.example.api.domain.Item;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;

public interface ItemRepository extends JpaRepository<Item, Long> {
    @Query("""
        select item from Item item
        where (:name is null or lower(item.name) like lower(concat('%', :name, '%')))
        """)
    Page<Item> findAllByName(String name, Pageable pageable);

    boolean existsByCode(String code);
}
